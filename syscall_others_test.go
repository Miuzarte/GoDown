//go:build !windows

package main

import (
	"testing"
)

func TestGetDownloadsFolder(t *testing.T) {
	t.Log(GetDownloadsFolder())
}
