package main

import (
	"syscall"
	"unsafe"

	log "github.com/sirupsen/logrus"
)

func init() {
	const DIV = 1024 * 1024 * 1024 // GB

	memStatus := &MEMORYSTATUSEX{}
	GlobalMemoryStatusEx(memStatus)
	memUsable := float64(memStatus.AvailPhys)
	if memUsable < memUsage {
		log.Fatalf("Not enough memory, %.2fGiB available", memUsable/DIV)
	}

	if DownloadsFolder = GetDownloadsFolder(); DownloadsFolder == "" {
		log.Fatalf("Failed to get downloads folder, please set it manually")
	}
}

func GetDownloadsFolder() string {
	// ::{374DE290-123F-4565-9164-39C4925E467B}
	downloadsClsid := syscall.GUID{Data1: 0x374DE290, Data2: 0x123F, Data3: 0x4565, Data4: [8]byte{0x91, 0x64, 0x39, 0xC4, 0x92, 0x5E, 0x46, 0x7B}}
	var pszPath uintptr
	err := SHGetKnownFolderPath(&downloadsClsid, 0, 0, &pszPath)
	if err != nil {
		log.Fatalf("Failed to get downloads folder: %v", err)
	}
	defer CoTaskMemFree(pszPath)
	return syscall.UTF16ToString((*[syscall.MAX_PATH]uint16)(unsafe.Pointer(pszPath))[:])
}
