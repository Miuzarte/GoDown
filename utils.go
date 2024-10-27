package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func FormatBytes(bytes int) string {
	const (
		B   = 1
		KiB = B * 1024
		MiB = KiB * 1024
		GiB = MiB * 1024
		TiB = GiB * 1024
		PiB = TiB * 1024
		EiB = PiB * 1024
	)
	switch {
	case bytes >= EiB:
		return fmt.Sprintf("%.2f EiB", float64(bytes)/EiB)
	case bytes >= PiB:
		return fmt.Sprintf("%.2f PiB", float64(bytes)/PiB)
	case bytes >= TiB:
		return fmt.Sprintf("%.2f TiB", float64(bytes)/TiB)
	case bytes >= GiB:
		return fmt.Sprintf("%.2f GiB", float64(bytes)/GiB)
	case bytes >= MiB:
		return fmt.Sprintf("%.2f MiB", float64(bytes)/MiB)
	case bytes >= KiB:
		return fmt.Sprintf("%.2f KiB", float64(bytes)/KiB)
	case bytes >= B+1:
		return fmt.Sprintf("%d Bytes", bytes)
	default:
		return fmt.Sprintf("%d Byte", bytes)
	}
}

func GetUniqueFilePath(path string) string {
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)

	uniquePath := path
	for i := 1; ; i++ {
		if _, err := os.Stat(uniquePath); os.IsNotExist(err) {
			break
		}
		uniquePath = filepath.Join(dir, fmt.Sprintf("%s(%d)%s", base, i, ext))
	}
	return uniquePath
}

func Hyperlink(link string) string {
	return fmt.Sprintf("\x1b]8;;file://%s\x1b\\%s\x1b]8;;\x1b\\", link, link)
}

type Limiter struct {
	Max int
	sem chan struct{}
}

func NewLimiter(max int) *Limiter {
	return &Limiter{
		Max: max,
		sem: make(chan struct{}, max),
	}
}

func (l *Limiter) Acquire() {
	l.sem <- struct{}{}
}

func (l *Limiter) Release() {
	<-l.sem
}
