package activity

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)

func TestMouseDistanceAccumulatesBetweenPositions(t *testing.T) {
	a := NewAggregator(DefaultIdleThreshold)

	a.MousePosition(t0, 0, 0)
	a.MousePosition(t0, 3, 4)  // +5
	a.MousePosition(t0, 3, 14) // +10

	got := a.Current()
	if got.MouseDistancePx != 15 {
		t.Fatalf("distance = %d, want 15", got.MouseDistancePx)
	}
}

func TestCountsClicksScrollsAndKeystrokes(t *testing.T) {
	a := NewAggregator(DefaultIdleThreshold)

	a.Click(t0)
	a.Click(t0)
	a.Scroll(t0)
	a.Keystroke(t0)
	a.Keystroke(t0)
	a.Keystroke(t0)

	got := a.Current()
	if got.Clicks != 2 || got.Scrolls != 1 || got.Keystrokes != 3 {
		t.Fatalf("got clicks=%d scrolls=%d keys=%d, want 2/1/3", got.Clicks, got.Scrolls, got.Keystrokes)
	}
}

func TestSampleSplitsActiveAndIdleSeconds(t *testing.T) {
	a := NewAggregator(DefaultIdleThreshold)

	a.Sample(t0, 0, "code.exe")                                       // active
	a.Sample(t0.Add(1*time.Second), 500*time.Millisecond, "code.exe") // active
	a.Sample(t0.Add(2*time.Second), 3*time.Second, "code.exe")        // no input this second

	got := a.Current()
	if got.ActiveSeconds != 2 || got.IdleSeconds != 1 {
		t.Fatalf("active=%d idle=%d, want 2/1", got.ActiveSeconds, got.IdleSeconds)
	}
}

func TestAppSecondsStopAfterIdleThreshold(t *testing.T) {
	a := NewAggregator(2 * time.Minute)

	a.Sample(t0, 0, "code.exe")
	a.Sample(t0.Add(time.Second), 90*time.Second, "code.exe")    // idle but under threshold: still counts
	a.Sample(t0.Add(2*time.Second), 3*time.Minute, "chrome.exe") // away from desk: not counted

	got := a.Current()
	if got.AppSeconds["code.exe"] != 2 {
		t.Fatalf("code.exe = %d, want 2", got.AppSeconds["code.exe"])
	}
	if _, ok := got.AppSeconds["chrome.exe"]; ok {
		t.Fatalf("chrome.exe should not be counted while away")
	}
}

func TestMinuteRolloverReturnsCompletedBucket(t *testing.T) {
	a := NewAggregator(DefaultIdleThreshold)

	a.Click(t0.Add(10 * time.Second))
	a.Click(t0.Add(70 * time.Second)) // next minute

	done := a.TakeCompleted()
	if len(done) != 1 {
		t.Fatalf("completed = %d, want 1", len(done))
	}
	if !done[0].Start.Equal(t0) || done[0].Clicks != 1 {
		t.Fatalf("completed bucket = %+v", done[0])
	}
	if a.Current().Clicks != 1 || !a.Current().Start.Equal(t0.Add(time.Minute)) {
		t.Fatalf("current bucket = %+v", a.Current())
	}
	if len(a.TakeCompleted()) != 0 {
		t.Fatalf("TakeCompleted should drain")
	}
}

func TestFlushClosesCurrentBucket(t *testing.T) {
	a := NewAggregator(DefaultIdleThreshold)
	a.Keystroke(t0)

	done := a.Flush()
	if len(done) != 1 || done[0].Keystrokes != 1 {
		t.Fatalf("flush = %+v", done)
	}
	if len(a.Flush()) != 0 {
		t.Fatalf("second flush should be empty")
	}
}

func TestCurrentReturnsCopy(t *testing.T) {
	a := NewAggregator(DefaultIdleThreshold)
	a.Sample(t0, 0, "code.exe")

	snap := a.Current()
	snap.AppSeconds["code.exe"] = 999

	if a.Current().AppSeconds["code.exe"] != 1 {
		t.Fatalf("mutating snapshot changed aggregator state")
	}
}
