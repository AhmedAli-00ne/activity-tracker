package queue

import (
	"path/filepath"
	"testing"
	"time"

	"activityagent/internal/activity"
)

func TestAppendThenReadAllRoundTrips(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "queue.jsonl"))
	start := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	recs := []Record{
		{Device: "LAPTOP-1", User: "sara", Bucket: activity.Bucket{Start: start, Clicks: 4, AppSeconds: map[string]int{"code.exe": 30}}},
		{Device: "LAPTOP-1", User: "sara", Bucket: activity.Bucket{Start: start.Add(time.Minute), Keystrokes: 9}},
	}

	if err := q.Append(recs[:1]); err != nil {
		t.Fatal(err)
	}
	if err := q.Append(recs[1:]); err != nil {
		t.Fatal(err)
	}

	got, err := q.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Bucket.Clicks != 4 || got[0].Bucket.AppSeconds["code.exe"] != 30 || got[1].Bucket.Keystrokes != 9 {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestReadAllOnMissingFileIsEmpty(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "nope.jsonl"))
	got, err := q.ReadAll()
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestAppendNothingIsNoop(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "queue.jsonl"))
	if err := q.Append(nil); err != nil {
		t.Fatal(err)
	}
}
