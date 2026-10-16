//go:build windows

package winput

import (
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type lastInputInfo struct {
	cbSize uint32
	dwTime uint32
}

// IdleDuration is the time since the last keyboard or mouse input.
func IdleDuration() time.Duration {
	info := lastInputInfo{cbSize: uint32(unsafe.Sizeof(lastInputInfo{}))}
	if r, _, _ := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&info))); r == 0 {
		return 0
	}
	now, _, _ := procGetTickCount.Call()
	// uint32 subtraction stays correct when the tick counter wraps (~49 days).
	return time.Duration(uint32(now)-info.dwTime) * time.Millisecond
}

// ForegroundApp returns the executable name of the program in front, in
// lower case (for example "excel.exe"), or "" when it cannot be read.
// Window titles are deliberately not read.
func ForegroundApp() string {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return ""
	}
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return ""
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)

	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return strings.ToLower(filepath.Base(windows.UTF16ToString(buf[:size])))
}

// SingleInstance returns false if another copy of the agent is already
// running for this user. The returned handle must stay open while running.
func SingleInstance(name string) (windows.Handle, bool, error) {
	n, err := windows.UTF16PtrFromString(`Local\` + name)
	if err != nil {
		return 0, false, err
	}
	h, err := windows.CreateMutex(nil, false, n)
	if err == windows.ERROR_ALREADY_EXISTS {
		windows.CloseHandle(h)
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return h, true, nil
}

const (
	mbOK              = 0x0
	mbOKCancel        = 0x1
	mbIconInformation = 0x40
	mbSetForeground   = 0x10000
	mbTopmost         = 0x40000
	idOK              = 1
)

// Confirm shows an OK/Cancel box and reports whether OK was pressed.
func Confirm(title, text string) bool {
	return messageBox(title, text, mbOKCancel) == idOK
}

// Info shows a box with a single OK button.
func Info(title, text string) {
	messageBox(title, text, mbOK)
}

func messageBox(title, text string, kind uintptr) uintptr {
	t, _ := windows.UTF16PtrFromString(title)
	m, _ := windows.UTF16PtrFromString(text)
	r, _, _ := procMessageBoxW.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)),
		kind|mbIconInformation|mbSetForeground|mbTopmost)
	return r
}
