//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strings"
)

// GetDownloadsFolder return executable dir, fallback work dir
func GetDownloadsFolder() string {
	exePath, err := os.Executable()
	if err != nil || strings.Contains(exePath, "go-build") {
		wd, err := os.Getwd()
		if err != nil {
			return os.Getenv("HOME")
		}
		return wd
	}
	return filepath.Dir(exePath)
}
