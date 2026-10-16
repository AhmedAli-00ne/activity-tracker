// Package queue stores finished activity buckets on disk until they are
// uploaded, so nothing is lost if the laptop is offline or restarts.
package queue

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"activityagent/internal/activity"
)

// Record is one minute of activity tagged with who and where.
type Record struct {
	Device string          `json:"device"`
	User   string          `json:"user"`
	Bucket activity.Bucket `json:"bucket"`
}

// Queue is an append-only JSON Lines file.
type Queue struct {
	mu   sync.Mutex
	path string
}

func New(path string) *Queue {
	return &Queue{path: path}
}

func (q *Queue) Append(recs []Record) error {
	if len(recs) == 0 {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(q.path), 0o700); err != nil {
		return fmt.Errorf("create queue dir: %w", err)
	}
	f, err := os.OpenFile(q.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open queue: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, r := range recs {
		if err := enc.Encode(r); err != nil {
			return fmt.Errorf("write queue record: %w", err)
		}
	}
	return nil
}

func (q *Queue) ReadAll() ([]Record, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	f, err := os.Open(q.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open queue: %w", err)
	}
	defer f.Close()

	var out []Record
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var r Record
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("parse queue line %d: %w", len(out)+1, err)
		}
		out = append(out, r)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read queue: %w", err)
	}
	return out, nil
}
