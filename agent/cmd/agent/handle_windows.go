//go:build windows

package main

import "golang.org/x/sys/windows"

func closeHandle(h windows.Handle) { _ = windows.CloseHandle(h) }
