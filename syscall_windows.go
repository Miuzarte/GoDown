//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"

	log "github.com/sirupsen/logrus"
)

var (
	Kernel32Dll = syscall.NewLazyDLL("kernel32.dll")
	shell32Dll  = syscall.NewLazyDLL("shell32.dll")
	ole32Dll    = syscall.NewLazyDLL("ole32.dll")
)

var (
	GlobalMemoryStatusExFunc = Kernel32Dll.NewProc("GlobalMemoryStatusEx")
	SHGetKnownFolderPathFunc = shell32Dll.NewProc("SHGetKnownFolderPath")
	CoTaskMemFreeFunc        = ole32Dll.NewProc("CoTaskMemFree")
)

// var memUsage float64 = 1024 * 1024 * 1024 // 内存使用量, 1GB

func GetMemoryStatus() *MEMORYSTATUSEX {
	memStatus := &MEMORYSTATUSEX{}
	GlobalMemoryStatusEx(memStatus)
	return memStatus
}

func GlobalMemoryStatusEx(memStatus *MEMORYSTATUSEX) (err error) {
	defer syscall.FreeLibrary(syscall.Handle(Kernel32Dll.Handle()))

	memStatus.Init()
	ret, _, err := GlobalMemoryStatusExFunc.Call(uintptr(unsafe.Pointer(memStatus)))
	if ret == 0 {
		return err
	}
	return nil
}

type MEMORYSTATUSEX struct {
	length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func (m MEMORYSTATUSEX) String() string {
	return fmt.Sprintf(`MEMORYSTATUSEX struct {
	MemoryLoad: %d %%,
	TotalPhys: %s,
	AvailPhys: %s,
	TotalPageFile: %s,
	AvailPageFile: %s,
	TotalVirtual: %s,
	AvailVirtual: %s,
	AvailExtendedVirtual: %s,
}`,
		m.MemoryLoad,
		FormatBytes(int(m.TotalPhys)),
		FormatBytes(int(m.AvailPhys)),
		FormatBytes(int(m.TotalPageFile)),
		FormatBytes(int(m.AvailPageFile)),
		FormatBytes(int(m.TotalVirtual)),
		FormatBytes(int(m.AvailVirtual)),
		FormatBytes(int(m.AvailExtendedVirtual)),
	)
}

func (m *MEMORYSTATUSEX) Init() {
	m.length = 64
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

func SHGetKnownFolderPath(rfid *syscall.GUID, dwFlags uint32, hToken syscall.Handle, pszPath *uintptr) (err error) {
	defer syscall.FreeLibrary(syscall.Handle(shell32Dll.Handle()))

	ret, _, err := SHGetKnownFolderPathFunc.Call(
		uintptr(unsafe.Pointer(rfid)),
		uintptr(dwFlags),
		uintptr(hToken),
		uintptr(unsafe.Pointer(pszPath)),
	)
	if ret != 0 {
		return err
	}
	return nil
}

func CoTaskMemFree(pv uintptr) {
	CoTaskMemFreeFunc.Call(pv)
}
