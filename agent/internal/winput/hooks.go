//go:build windows

package winput

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Sink receives input events. activity.Aggregator satisfies it.
type Sink interface {
	MousePosition(now time.Time, x, y int)
	Click(now time.Time)
	Scroll(now time.Time)
	Keystroke(now time.Time)
}

type point struct{ X, Y int32 }

// msllHookStruct mirrors MSLLHOOKSTRUCT; only the cursor position is read.
type msllHookStruct struct {
	Pt point
	_  uint32 // mouseData
	_  uint32 // flags
	_  uint32 // time
	_  uintptr
}

// Hooks runs the low-level mouse and keyboard hooks on a dedicated OS thread.
type Hooks struct {
	threadID uint32
	done     chan error
}

// Start installs the hooks and returns once they are live.
func Start(sink Sink) (*Hooks, error) {
	h := &Hooks{done: make(chan error, 1)}
	ready := make(chan error, 1)
	go h.loop(sink, ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return h, nil
}

// Stop removes the hooks and waits for the hook thread to exit.
func (h *Hooks) Stop() error {
	procPostThreadMessageW.Call(uintptr(h.threadID), wmQuit, 0, 0)
	return <-h.done
}

func (h *Hooks) loop(sink Sink, ready chan<- error) {
	// Hooks are delivered to the thread that installed them, which must pump
	// messages, so this goroutine stays on one OS thread for its whole life.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	h.threadID = windows.GetCurrentThreadId()

	var mod windows.Handle
	if err := windows.GetModuleHandleEx(0, nil, &mod); err != nil {
		ready <- fmt.Errorf("get module handle: %w", err)
		return
	}

	mouseProc := syscall.NewCallback(func(nCode, wParam, lParam uintptr) uintptr {
		if int32(nCode) >= 0 {
			now := time.Now()
			switch wParam {
			case wmMouseMove:
				// lParam points at an MSLLHOOKSTRUCT owned by Windows for the
				// duration of this call; go vet's unsafe.Pointer warning here
				// is the standard false positive for syscall callbacks.
				m := (*msllHookStruct)(unsafe.Pointer(lParam))
				sink.MousePosition(now, int(m.Pt.X), int(m.Pt.Y))
			case wmLButtonDown, wmRButtonDown, wmMButtonDown, wmXButtonDown:
				sink.Click(now)
			case wmMouseWheel, wmMouseHWheel:
				sink.Scroll(now)
			}
		}
		r, _, _ := procCallNextHookEx.Call(0, nCode, wParam, lParam)
		return r
	})

	// Counts key releases only: one per physical press, no auto-repeat, and
	// the hook data (which key) is never read.
	keyProc := syscall.NewCallback(func(nCode, wParam, lParam uintptr) uintptr {
		if int32(nCode) >= 0 && (wParam == wmKeyUp || wParam == wmSysKeyUp) {
			sink.Keystroke(time.Now())
		}
		r, _, _ := procCallNextHookEx.Call(0, nCode, wParam, lParam)
		return r
	})

	mouseHook, _, err := procSetWindowsHookExW.Call(whMouseLL, mouseProc, uintptr(mod), 0)
	if mouseHook == 0 {
		ready <- fmt.Errorf("install mouse hook: %w", err)
		return
	}
	defer procUnhookWindowsHookEx.Call(mouseHook)

	keyHook, _, err := procSetWindowsHookExW.Call(whKeyboardLL, keyProc, uintptr(mod), 0)
	if keyHook == 0 {
		ready <- fmt.Errorf("install keyboard hook: %w", err)
		return
	}
	defer procUnhookWindowsHookEx.Call(keyHook)

	ready <- nil

	var msg [48]byte // MSG; contents unused
	for {
		r, _, err := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg[0])), 0, 0, 0)
		switch int32(r) {
		case 0: // WM_QUIT
			h.done <- nil
			return
		case -1:
			h.done <- fmt.Errorf("message loop: %w", err)
			return
		}
	}
}
