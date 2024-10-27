//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
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
