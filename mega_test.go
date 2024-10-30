package main

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"testing"
)

func TestMegaInfo(t *testing.T) {
	fileId := "0rASQYSR"

	fileKeyBase64 := urlB64ToStdB64("KD1y_pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4")
	// KD1y/pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4=
	fmt.Printf("fileKeyBase64: %v\n", fileKeyBase64)

	fileKey, err := base64.StdEncoding.DecodeString(fileKeyBase64)
	if err != nil {
		t.Fatalf("解码文件密钥失败: %v\n", err)
	}
	fmt.Printf("fileKey: %s\n", fileKey)

	downloadURL, size, at, err := getFileInfo(fileId)
	if err != nil {
		t.Fatalf("获取文件信息失败: %v\n", err)
	}
	fmt.Printf("下载 URL: %s\n", downloadURL)
	fmt.Printf("文件大小: %d\n", size)
	fmt.Printf("文件属性: %s\n", at)

	// key, err := hex.DecodeString(fileKey)
	// if err != nil {
	// 	t.Fatalf("解析文件密钥失败: %v\n", err)
	// }
	deAt, err := decryptAES128ECB(fileKey, []byte(at))
	if err != nil {
		t.Fatalf("解密文件属性失败: %v\n", err)
	}
	fmt.Printf("解密文件属性: %s\n", deAt)
}

func TestMegaUrl(t *testing.T) {
	u, err := url.Parse("https://mega.nz/file/0rASQYSR#KD1y_pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4")
	if err != nil {
		t.Fatalf("解析 URL 失败: %v\n", err)
	}
	fmt.Printf("Url: %#v\n", u)

	// uu := &url.URL{
	// 	Scheme:      "https",
	// 	Opaque:      "",
	// 	User:        (*url.Userinfo)(nil),
	// 	Host:        "mega.nz",
	// 	Path:        "/file/0rASQYSR",
	// 	RawPath:     "",
	// 	OmitHost:    false,
	// 	ForceQuery:  false,
	// 	RawQuery:    "",
	// 	Fragment:    "KD1y_pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4",
	// 	RawFragment: "",
	// }

	isMega := u.Host == "mega.nz" || u.Host == "mega.co.nz"
	fileId := path.Base(u.Path)
	fileKey := u.Fragment

	fmt.Printf("IsMega: %v\n", isMega)
	fmt.Printf("FileId: %s\n", fileId)
	fmt.Printf("FileKey: %s\n", fileKey)
}
