// Package consent holds the notice employees see on first run and remembers
// that they acknowledged it. Tracking never starts without that record.
package consent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PolicyVersion must be bumped whenever PolicyText changes, so everyone is
// shown the new wording.
const PolicyVersion = 1

const PolicyText = `This company laptop runs an activity tracker.

What it records, in one-minute totals:
  - how far the mouse moved, and how many clicks and scrolls
  - how many keys were pressed (a number only)
  - active and idle time
  - the name of the program in front (for example "excel.exe")

What it never records:
  - which keys you press or anything you have typed
  - passwords, messages, documents or files
  - what is on your screen (no screenshots)
  - window titles, web addresses or browsing history

The tray icon near the clock shows while it is running.
Questions: contact IT.`

// AcceptPrompt is appended to PolicyText on the first-run notice.
const AcceptPrompt = "Press OK to acknowledge and start, or Cancel to not start it now."

type record struct {
	Version    int       `json:"version"`
	User       string    `json:"user"`
	AcceptedAt time.Time `json:"acceptedAt"`
}

func path(dir string) string { return filepath.Join(dir, "consent.json") }

// HasAccepted reports whether this version of the policy was acknowledged.
func HasAccepted(dir string, version int) bool {
	data, err := os.ReadFile(path(dir))
	if err != nil {
		return false
	}
	var r record
	if err := json.Unmarshal(data, &r); err != nil {
		return false
	}
	return r.Version == version
}

func RecordAcceptance(dir string, version int, user string, now time.Time) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	data, err := json.MarshalIndent(record{Version: version, User: user, AcceptedAt: now.UTC()}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode consent: %w", err)
	}
	if err := os.WriteFile(path(dir), data, 0o600); err != nil {
		return fmt.Errorf("save consent: %w", err)
	}
	return nil
}
