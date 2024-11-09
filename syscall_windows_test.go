//go:build windows

package main

import (
	"fmt"
	"runtime"
	"testing"
)

func TestGlobalMemoryStatusEx(t *testing.T) {
	memStatsBefore := &runtime.MemStats{}
	runtime.ReadMemStats(memStatsBefore)

	memStatus := &MEMORYSTATUSEX{}
	err := GlobalMemoryStatusEx(memStatus)
	if err != nil {
		t.Fatalf("GlobalMemoryStatusEx failed: %v", err)
	}
	fmt.Println(memStatus)

	memStatsAfter := &runtime.MemStats{}
	runtime.ReadMemStats(memStatsAfter)

	fmt.Printf("Memory usage before: %d bytes\n", memStatsBefore.Alloc)
	fmt.Printf("Memory usage after: %d bytes\n", memStatsAfter.Alloc)
}

func TestGetDownloadsFolder(t *testing.T) {
	t.Log(GetDownloadsFolder())
}
