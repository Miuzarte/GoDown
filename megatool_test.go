package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestBase64UrlDecode(t *testing.T) {
	urlKey1, err := base64UrlDecode("KD1y_pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("urlKey1: %s\n", urlKey1)
	t.Logf("len(urlKey1): %v\n", len(urlKey1))
	urlKey2, err := base64urldecode("KD1y_pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("urlKey2: %s\n", urlKey2)
	t.Logf("len(urlKey2): %v\n", len(urlKey2))

	fmt.Printf("bytes.Equal(urlKey1, urlKey2): %v\n", bytes.Equal(urlKey1, urlKey2))
}

func base64urldecode(s string) ([]byte, error) {
	enc := base64.RawURLEncoding
	// mega base64 decoder accepts the characters from both URLEncoding and StdEncoding
	// though nearly all strings are URL encoded
	s = strings.Replace(s, "+", "-", -1)
	s = strings.Replace(s, "/", "_", -1)
	return enc.DecodeString(s)
}

func TestUnpackKey(t *testing.T) {
	// urlKey, err := base64UrlDecode("KD1y_pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4")
	urlKey, err := base64UrlDecode("LbCyh8TYNapGniQGk_723imBeUEafjKw0uL3sKjTLjo")
	if err != nil {
		t.Fatal(err)
	}
	if len(urlKey) != 32 {
		t.Fatal("len(urlKey) != 32")
	}

	// at := "CVHa0S-CPMZiBtn4g_5ebCkj-oV92Xob0INPSzoAoT-jgz9nVZjuW6Eind3vUnz43h9YMnwfMo7KOjQRz6ezvg"
	at := "yx4Xbs14ovw0UqQf0u6_gdIkx-EEGvhtovhsvD-WArHxcHFzum1OhP4F-MDEemYryFRrp6-XOnfZQhZNsCfHQA"

	aesKey, _, _ := unpackKey(urlKey)

	attr, err := decryptAttr(aesKey, at)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("attr: %v\n", attr)

}

func TestParseLink(t *testing.T) {
	url := "https://mega.nz/file/0rASQYSR#KD1y_pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4"

	for _, lr := range linkRegexes {
		match := lr.re.MatchString(url)
		if match {
			matches := lr.re.FindAllStringSubmatch(url, -1)
			if len(matches) > 0 {
				fmt.Printf("matches: %#v\n", matches)
			}
		}
	}
}

func TestMegaDownloadWithDecrypt(t *testing.T) {
	// link := parseLink("https://mega.nz/file/0rASQYSR#KD1y_pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4")
	// if link == nil {
	// 	t.Fatal("parseLink() failed")
	// }

	s := NewMegaSession()

	os.Remove("A:/Git/GoDown/GoDown_random.bin")
	f, err := os.OpenFile("A:/Git/GoDown/GoDown_random.bin", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err = s.Download(f, "0rASQYSR", "KD1y_pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4")
	if err != nil {
		os.Remove("A:/Git/GoDown/GoDown_random.bin")
		t.Fatal(err)
	}
}

func TestPrepareDownload(t *testing.T) {
	s := NewMegaSession()
	params, err := s.prepareDownload("0rASQYSR", "KD1y_pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("params: %#v\n", *params)
}
