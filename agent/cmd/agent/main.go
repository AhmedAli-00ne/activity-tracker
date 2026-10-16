//go:build windows

// Command agent is the visible activity tracker that runs on company laptops.
//
// On first run it shows the policy notice and does nothing unless the person
// presses OK. While running it shows a tray icon and writes one-minute
// activity totals to a local queue for upload.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"fyne.io/systray"

	"activityagent/internal/activity"
	"activityagent/internal/consent"
	"activityagent/internal/queue"
	"activityagent/internal/trayicon"
	"activityagent/internal/winput"
)

const appName = "Activity Tracker"

func main() {
	dataDir := flag.String("data-dir", defaultDataDir(), "where consent, queue and log files are kept")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0o700); err != nil {
		winput.Info(appName, fmt.Sprintf("Cannot create %s: %v", *dataDir, err))
		os.Exit(1)
	}
	logFile, err := os.OpenFile(filepath.Join(*dataDir, "agent.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	mutex, first, err := winput.SingleInstance("ActivityTrackerAgent")
	if err != nil {
		log.Fatalf("single instance check: %v", err)
	}
	if !first {
		return
	}
	defer closeHandle(mutex)

	userName := currentUser()
	if !consent.HasAccepted(*dataDir, consent.PolicyVersion) {
		if !winput.Confirm(appName, consent.PolicyText+"\n\n"+consent.AcceptPrompt) {
			log.Printf("policy v%d not acknowledged by %s; not starting", consent.PolicyVersion, userName)
			return
		}
		if err := consent.RecordAcceptance(*dataDir, consent.PolicyVersion, userName, time.Now()); err != nil {
			log.Fatalf("record consent: %v", err)
		}
	}

	run(*dataDir, userName)
}

func run(dataDir, userName string) {
	device, _ := os.Hostname()
	agg := activity.NewAggregator(activity.DefaultIdleThreshold)
	q := queue.New(filepath.Join(dataDir, "queue.jsonl"))

	save := func(buckets []activity.Bucket) {
		if len(buckets) == 0 {
			return
		}
		recs := make([]queue.Record, 0, len(buckets))
		for _, b := range buckets {
			recs = append(recs, queue.Record{Device: device, User: userName, Bucket: b})
		}
		if err := q.Append(recs); err != nil {
			log.Printf("save activity: %v", err)
		}
	}

	hooks, err := winput.Start(agg)
	if err != nil {
		log.Fatalf("start input hooks: %v", err)
	}

	stopSampler := make(chan struct{})
	samplerDone := make(chan struct{})
	go func() {
		defer close(samplerDone)
		tick := time.NewTicker(activity.SampleInterval)
		defer tick.Stop()
		for {
			select {
			case <-stopSampler:
				return
			case now := <-tick.C:
				agg.Sample(now, winput.IdleDuration(), winput.ForegroundApp())
				save(agg.TakeCompleted())
			}
		}
	}()

	log.Printf("started for %s on %s", userName, device)

	systray.Run(func() {
		systray.SetIcon(trayicon.ICO())
		systray.SetTitle(appName)
		systray.SetTooltip(appName + " is on")

		status := systray.AddMenuItem(appName+" is on", "")
		status.Disable()
		policy := systray.AddMenuItem("What is recorded…", "Show what this tracker collects")
		go func() {
			for range policy.ClickedCh {
				winput.Info(appName, consent.PolicyText)
			}
		}()
	}, func() {
		close(stopSampler)
		<-samplerDone
		if err := hooks.Stop(); err != nil {
			log.Printf("stop hooks: %v", err)
		}
		save(agg.Flush())
		log.Printf("stopped")
	})
}

func defaultDataDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "ActivityTracker")
}

func currentUser() string {
	u, err := user.Current()
	if err != nil {
		return os.Getenv("USERNAME")
	}
	return u.Username
}
