package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"
)

func TestStartSession(t *testing.T) {
	s, err := startSession(SESSION_OPEN | SESSION_AUTH_ONLY | SESSION_AUTH_OPTIONAL)
	if err != nil {
		t.Fatalf("startSession() failed: %v", err)
	}
	fmt.Printf("s: %#v\n", s)

	_ = &MegaSession{
		http:            &http.Client{},
		maxUL:           0,
		maxDL:           0,
		proxy:           "",
		maxWorkers:      0,
		id:              1730129107,
		sid:             "",
		rid:             "p8jkvh47b5",
		apiURLParams:    map[string]string{},
		passwordSaltV2:  "",
		passwordKey:     []uint8(nil),
		passwordKeySave: []uint8(nil),
		masterKey:       []uint8(nil),
		userHandle:      "",
		userName:        "",
		userEmail:       "",
		shareKeys:       map[string]string{},
		statusUserdata:  interface{}(nil),
		lastRefresh:     0,
		createPreview:   true,
		resumeEnabled:   true,
	}
}

func TestMegaFileAttr(t *testing.T) {
	s, err := startSession(SESSION_OPEN | SESSION_AUTH_ONLY | SESSION_AUTH_OPTIONAL)
	if err != nil {
		t.Fatalf("startSession() failed: %v", err)
	}

	// ml := parseLink("https://mega.nz/file/0rASQYSR#KD1y_pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4")
	// if ml == nil {
	// 	t.Fatalf("parseLink() failed")
	// }

	// dlDataParams, err := s.prepareDownload(ml.Handle, ml.Key)
	// dlDataParams, err := s.prepareDownload("0rASQYSR", "KD1y_pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4")
	dlDataParams, err := s.prepareDownload("pgcDAZLT", "LbCyh8TYNapGniQGk_723imBeUEafjKw0uL3sKjTLjo")
	if err != nil {
		t.Fatal(err)
	}
	/**
	type MegaDownloadDataParams struct {
		nodeKey     []byte
		downloadUrl string
		nodeHandle  string
		nodeName    string
		nodeSize    uint64
	}
	**/
	fmt.Printf("dlDataParams.nodeKey: %s\n", dlDataParams.nodeKey)
	fmt.Printf("dlDataParams.downloadUrl: %s\n", dlDataParams.downloadUrl)
	fmt.Printf("dlDataParams.nodeHandle: %s\n", dlDataParams.nodeHandle)
	fmt.Printf("dlDataParams.nodeName: %s\n", dlDataParams.nodeName)
	fmt.Printf("dlDataParams.nodeSize: %d\n", dlDataParams.nodeSize)

}

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

	aesKey := make([]byte, 16)
	unpackKey(urlKey, &aesKey, nil, nil)

	attr, err := decryptAttr(aesKey, at)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("attr: %v\n", attr)

}

func TestParseLink(t *testing.T) {
	re := regexp.MustCompile("^https?://mega\\.nz/file/([a-z0-9_-]{8})#([a-z0-9_-]{43}={0,2})$")
	url := "https://mega.nz/file/0rASQYSR#KD1y_pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4"

	match := re.MatchString(url)
	fmt.Printf("match: %#v\n", match)

	matches1 := re.FindAllStringSubmatch(url, -1)
	fmt.Printf("matches1: %#v\n", matches1)

	matches2 := re.FindAllString(url, -1)
	fmt.Printf("matches2: %#v\n", matches2)

	// for _, lr := range linkRegexes {
	// 	match := lr.re.MatchString(url)
	// 	if match {
	// 		matches := lr.re.FindStringSubmatch(url)
	// 		if len(matches) > 0 {
	// 			fmt.Printf("matches: %#v\n", matches)
	// 		}
	// 	}
	// }
}

func TestMegaDownloadWithDecrypt(t *testing.T) {
	// link := parseLink("https://mega.nz/file/0rASQYSR#KD1y_pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4")
	// if link == nil {
	// 	t.Fatal("parseLink() failed")
	// }

	s := newMegaSession()

	os.Remove("A:/Git/GoDown/GoDown_random.bin")
	f, err := os.OpenFile("A:/Git/GoDown/GoDown_random.bin", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err = s.download("0rASQYSR", "KD1y_pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4", f)
	if err != nil {
		os.Remove("A:/Git/GoDown/GoDown_random.bin")
		t.Fatal(err)
	}
}

func TestPrepareDownload(t *testing.T) {
	s := newMegaSession()
	params, err := s.prepareDownload("0rASQYSR", "KD1y_pMnRAJkgp1sPtcno5L548L1WJcfQhN0SCITuI4")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("params: %#v\n", *params)
}
