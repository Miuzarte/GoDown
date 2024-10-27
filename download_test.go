package main

import (
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"testing"
)

func TestGetHeader(t *testing.T) {
	j := &Job{
		Url: "https://pkg.biligame.com/games/mrfz_2.3.81_20241002_113301_738f9.apk",
	}

	req, err := http.NewRequest("HEAD", j.Url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := NewClient().Do(req)
	if err != nil {
		t.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("resp.StatusCode: %v\n", resp.StatusCode)
	fmt.Printf("resp.Header: %v\n", resp.Header)

	contentDisposition := resp.Header.Get("Content-Disposition")
	fmt.Printf("contentDisposition: %v\n", contentDisposition)

	contentLength := resp.Header.Get("Content-Length")
	if contentLength != "" {
		fmt.Println("Content-Length:", contentLength)
	} else {
		fmt.Println("Content-Length header not found")
	}
}

func TestInit(t *testing.T) {
	j := &Job{Url: "https://pkg.biligame.com/games/bhxqtd_2.6.0_20241012_105217_37c02.apk"}
	j.init()
	fmt.Println(j)
	for i, block := range j.Blocks {
		fmt.Printf("Block %d: bytes=%d-%d\n", i, block.start, block.end)
	}
}

func TestFullDownlaod(t *testing.T) {
	j := &Job{Url: "https://pkg.biligame.com/games/bhxqtd_2.6.0_20241012_105217_37c02.apk"}
	j.init()
	fmt.Println(j)
	j.Start()
}

func TestEnvProxy(t *testing.T) {
	j := &Job{Url: "https://pkg.biligame.com/games/bhxqtd_2.6.0_20241012_105217_37c02.apk"}
	req, err := http.NewRequest("HEAD", j.Url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	proxy, err := http.ProxyFromEnvironment(req)
	if err != nil {
		t.Fatalf("Failed to get proxy: %v", err)
	}
	fmt.Println(proxy)
}

func TestRange(t *testing.T) {
	j := &Job{Url: "https://pkg.biligame.com/games/bhxqtd_2.6.0_20241012_105217_37c02.apk"}
	j.init()
	fmt.Println(j)

	fmt.Println(j.size)
	fmt.Println(blockSize)
	// 打印第1, 2，len-1, 块的范围
	b0 := j.Blocks[0]
	b1 := j.Blocks[1]
	l := len(j.Blocks)
	b_l2 := j.Blocks[l-2]
	b_l1 := j.Blocks[l-1]
	fmt.Printf("Block 0: bytes=%d-%d, size: %d\n", b0.start, b0.end, b0.end-b0.start+1)
	fmt.Printf("Block 1: bytes=%d-%d, size: %d\n", b1.start, b1.end, b1.end-b1.start+1)
	fmt.Printf("Block %d: bytes=%d-%d, size: %d\n", l-2, b_l2.start, b_l2.end, b_l2.end-b_l2.start+1)
	fmt.Printf("Block %d: bytes=%d-%d, size: %d\n", l-1, b_l1.start, b_l1.end, b_l1.end-b_l1.start+1)
}

func TestGetUniqueFilePath(t *testing.T) {
	fmt.Println(GetUniqueFilePath(`D:\Miuzarte\Downloads\bhxqtd_2.6.0_20241012_105217_37c02.apk`))
	fmt.Println(GetUniqueFilePath(`D:\Miuzarte\Downloads/bhxqtd_2.6.0_20241012_105217_37c02.apk`))
}

func TestPathJoin(t *testing.T) {
	fmt.Println(path.Join(`D:\Miuzarte\Downloads`, `bhxqtd_2.6.0_20241012_105217_37c02.apk`))
	fmt.Println(filepath.Join(`D:\Miuzarte\Downloads`, `bhxqtd_2.6.0_20241012_105217_37c02.apk`))
}
