# Activity Tracker

A visible, Hubstaff-style activity tracker for company Windows laptops,
requested by IT. Employees are told about it, see a tray icon while it
runs, and must acknowledge a notice before anything is recorded.

## Status

| Part | State |
|------|-------|
| Windows agent (`agent/`) | Built and tested. Records activity locally. |
| Uploading to a server | Not built yet |
| Server and admin dashboard | Not built yet |
| Installer for mass deployment | Not built yet |

## What the agent records

In one-minute totals, written to a local queue file:

- mouse distance in pixels, clicks, scrolls
- number of keys pressed (a count only)
- active and idle seconds (a second is active if there was any input)
- seconds per foreground program, by exe name (for example `excel.exe`),
  not counted after 3 minutes without input

What it never records: which keys were pressed or anything typed, window
titles, web addresses, screenshots, files, passwords.

Keystrokes are counted on key *release*, so holding a key down counts once,
and the code never reads which key it was.

## Set up a fresh laptop

Needs Windows 10/11 with winget (built in on Windows 11). From the repo
root:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\setup-dev.ps1
```

This installs Git, Go and Node.js LTS if missing, then runs the agent
tests and builds the exe. Re-running is safe. Open a new terminal
afterwards so every tool is on PATH.

## Build, test, run

```powershell
cd agent
go test -cover ./...
go build -trimpath -ldflags "-H windowsgui -s -w" -o bin\ActivityTracker.exe ./cmd/agent
.\bin\ActivityTracker.exe
```

`-H windowsgui` hides the console window; the tray icon is the UI.

On first run the agent shows the policy notice. Cancel exits without
recording anything. OK records the acknowledgement and starts tracking.

| Flag | Default | Purpose |
|------|---------|---------|
| `-data-dir` | `%LOCALAPPDATA%\ActivityTracker` | Where the files below are kept |

Files in the data dir:

| File | Contents |
|------|----------|
| `consent.json` | Policy version acknowledged, by whom, when |
| `queue.jsonl` | One JSON line per finished minute |
| `agent.log` | Start/stop and errors |

Only one copy runs per Windows user. Starting it again does nothing.
To stop it, end `ActivityTracker.exe` in Task Manager; there is
deliberately no Quit item in the tray.

To try it without the notice dialog (for example in automated tests),
point `-data-dir` at a folder that already has a `consent.json` with
`"version": 1`.

## Code layout

```
agent/
  cmd/agent/           entry point: notice, single instance, sampler, tray
  internal/activity/   turns input events into one-minute buckets (pure, tested)
  internal/queue/      JSON Lines queue on disk (tested)
  internal/consent/    notice text and acknowledgement record (tested)
  internal/trayicon/   tray icon drawn in code (tested)
  internal/winput/     Windows APIs: input hooks, idle time, foreground exe,
                       single-instance mutex, message boxes
scripts/
  setup-dev.ps1        toolchain setup for a fresh laptop
```

`internal/winput` and `cmd/agent` are Windows-only and have no unit tests;
they were checked with a live 75-second run that produced a real bucket.

### Changing the notice

The notice text is `PolicyText` in `agent/internal/consent/consent.go`.
If you change it, bump `PolicyVersion` so everyone is shown the new text
and acknowledges it again.

## Known gaps

- No upload yet: data stays in each laptop's `queue.jsonl`.
- No start-at-login: run the exe manually until an installer exists.
- The exe is unsigned, so Windows SmartScreen may warn on first launch.
- `go vet` reports "possible misuse of unsafe.Pointer" in
  `internal/winput/hooks.go`. It's the standard false positive for Windows
  hook callbacks and is commented in the code.
- `go test -race` needs cgo (a C compiler), which the setup script does
  not install.
