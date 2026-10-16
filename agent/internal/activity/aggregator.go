// Package activity turns raw input events into one-minute activity buckets.
//
// It only ever stores counts and durations. Which keys were pressed, what was
// typed and what is on screen are never passed in, so they cannot be stored.
package activity

import (
	"math"
	"sync"
	"time"
)

// SampleInterval is how often the agent calls Sample. A second counts as
// active when there was any input during it.
const SampleInterval = time.Second

// DefaultIdleThreshold is how long without input before the person is treated
// as away from the laptop, so foreground-app time stops counting.
const DefaultIdleThreshold = 3 * time.Minute

// Bucket is one minute of activity.
type Bucket struct {
	Start           time.Time      `json:"start"`
	MouseDistancePx int            `json:"mouseDistancePx"`
	Clicks          int            `json:"clicks"`
	Scrolls         int            `json:"scrolls"`
	Keystrokes      int            `json:"keystrokes"`
	ActiveSeconds   int            `json:"activeSeconds"`
	IdleSeconds     int            `json:"idleSeconds"`
	AppSeconds      map[string]int `json:"appSeconds"`
}

func (b Bucket) clone() Bucket {
	apps := make(map[string]int, len(b.AppSeconds))
	for k, v := range b.AppSeconds {
		apps[k] = v
	}
	b.AppSeconds = apps
	return b
}

// Aggregator is safe for concurrent use: input hooks and the sampling loop
// run on different goroutines.
type Aggregator struct {
	mu            sync.Mutex
	idleThreshold time.Duration
	current       *Bucket
	completed     []Bucket
	lastX, lastY  int
	hasLast       bool
}

func NewAggregator(idleThreshold time.Duration) *Aggregator {
	return &Aggregator{idleThreshold: idleThreshold}
}

// bucketAt returns the bucket for now's minute, closing the previous one if
// the minute has changed. Caller must hold mu.
func (a *Aggregator) bucketAt(now time.Time) *Bucket {
	start := now.Truncate(time.Minute)
	if a.current != nil && a.current.Start.Equal(start) {
		return a.current
	}
	if a.current != nil {
		a.completed = append(a.completed, *a.current)
	}
	a.current = &Bucket{Start: start, AppSeconds: map[string]int{}}
	return a.current
}

func (a *Aggregator) MousePosition(now time.Time, x, y int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	b := a.bucketAt(now)
	if a.hasLast {
		dx, dy := float64(x-a.lastX), float64(y-a.lastY)
		b.MouseDistancePx += int(math.Round(math.Hypot(dx, dy)))
	}
	a.lastX, a.lastY, a.hasLast = x, y, true
}

func (a *Aggregator) Click(now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.bucketAt(now).Clicks++
}

func (a *Aggregator) Scroll(now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.bucketAt(now).Scrolls++
}

// Keystroke records that a key was pressed. It deliberately takes no key.
func (a *Aggregator) Keystroke(now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.bucketAt(now).Keystrokes++
}

// Sample records one SampleInterval: idle is the time since the last input
// and app is the foreground program's executable name ("" if unknown).
func (a *Aggregator) Sample(now time.Time, idle time.Duration, app string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	b := a.bucketAt(now)
	if idle < SampleInterval {
		b.ActiveSeconds++
	} else {
		b.IdleSeconds++
	}
	if app != "" && idle < a.idleThreshold {
		b.AppSeconds[app]++
	}
}

// Current returns a copy of the in-progress bucket.
func (a *Aggregator) Current() Bucket {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.current == nil {
		return Bucket{}
	}
	return a.current.clone()
}

// TakeCompleted returns and clears the buckets whose minute has ended.
func (a *Aggregator) TakeCompleted() []Bucket {
	a.mu.Lock()
	defer a.mu.Unlock()
	done := a.completed
	a.completed = nil
	return done
}

// Flush closes the in-progress bucket and returns everything not yet taken.
// Used on shutdown.
func (a *Aggregator) Flush() []Bucket {
	a.mu.Lock()
	defer a.mu.Unlock()
	done := a.completed
	if a.current != nil {
		done = append(done, *a.current)
	}
	a.completed, a.current = nil, nil
	return done
}
