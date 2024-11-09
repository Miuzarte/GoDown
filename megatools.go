package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	API_URL = "https://g.api.mega.co.nz"

	RETRIES      = 10
	minSleepTime = 10 * time.Millisecond // for retries
	maxSleepTime = 5 * time.Second       // for retries
)

func DownloadMegaPublicFile(link string, dst io.Writer) error {
	s := NewMegaSession()

	l := parseLink(link)
	if l == nil {
		return fmt.Errorf("invalid link: %s", link)
	}

	switch l.Type {
	case LINK_FILE:
		return s.Download(dst, l.Handle, l.Key)
	case LINK_FOLDER:
		panic("Not implemented")
		return s.Download(dst, l.Handle, l.Key, l.Specific)
	default:
		panic("unreachable")
	}
}

type MegaSession struct {
	// http *http.Client
	// maxUL int
	// maxDL int
	// proxy string
	// maxWorkers int
	sn int64 // Sequence number, 自增
	// sid string
	// rid string
	// apiURLParams map[string]string
	// passwordSaltV2 string
	// passwordKey []byte
	// passwordKeySave []byte
	// masterKey []byte
	// rsaKey RSAKey
	// userHandle string
	// userName string
	// userEmail string
	// shareKeys map[string]string
	// fsNodes []FSNode
	// statusCallback
	// statusUserdata interface{}
	// lastRefresh int64
	// createPreview bool
	// resumeEnabled bool

	apiMu sync.Mutex // 序列化 api 请求
}

func NewMegaSession() *MegaSession {
	return &MegaSession{
		sn: time.Now().Unix(),
	}
}

type MegaDownloadDataParams struct {
	downloadUrl string
	nodeName    string
	nodeSize    uint64
	aesKey      []byte
	metaMacXor  []byte // 计算文件 MAC 用, 未实现
	nonce       []byte
}

func (s *MegaSession) Download(dst io.Writer, handle, key string, specific ...string) error {
	if len(specific) > 0 {
		panic("Not implemented")
	}
	params, err := s.prepareDownload(handle, key)
	if err != nil {
		return err
	}
	return s.downloadData(params, dst)
}

type MegaDownloadReq [1]struct {
	Cmd string `json:"a"`
	G   int    `json:"g"`
	SSL int    `json:"ssl,omitempty"`
	P   string `json:"p,omitempty"`
	N   string `json:"n,omitempty"` // hash
}

type MegaDownloadResp [1]struct {
	G    string   `json:"g"`
	Attr string   `json:"at"`
	Size uint64   `json:"s"`
	Err  ErrorMsg `json:"e"`
}

// prepareDownload 获取 链接, 大小, 属性(文件名), 解包密钥
func (s *MegaSession) prepareDownload(handle, key string) (*MegaDownloadDataParams, error) {
	req, err := json.Marshal(
		MegaDownloadReq{{
			Cmd: "g",
			G:   1,
			SSL: 0,
			P:   handle,
		}},
	)
	if err != nil {
		return nil, err
	}
	resp, err := s.apiRequest(req)
	if err != nil {
		return nil, err
	}
	var dlResp MegaDownloadResp
	err = json.Unmarshal(resp, &dlResp)
	if err != nil {
		return nil, err
	}

	url := dlResp[0].G
	at := dlResp[0].Attr
	size := dlResp[0].Size
	megaErr := dlResp[0].Err.Parse()
	if url == "" {
		return nil, fmt.Errorf("failed to determine download url")
	}
	if at == "" {
		return nil, fmt.Errorf("failed to get file attributes")
	}
	if size == 0 {
		return nil, fmt.Errorf("failed to determine file size")
	}
	if megaErr != nil {
		return nil, megaErr
	}

	// 解码节点密钥
	urlKey, err := base64UrlDecode(key)
	if err != nil {
		return nil, err
	}
	if len(urlKey) != 32 {
		return nil, fmt.Errorf("failed to retrieve file key")
	}

	// 初始化密钥
	aesKey, metaMacXor, nonce := unpackKey(urlKey)
	if len(aesKey) != 16 || len(metaMacXor) != 8 || len(nonce) != 8 {
		return nil, fmt.Errorf("failed to unpack file key")
	}
	// 解密属性
	attr, err := decryptAttr(aesKey, at)
	if err != nil {
		return nil, err
	}

	return &MegaDownloadDataParams{
		downloadUrl: url,
		nodeName:    attr.Name,
		nodeSize:    size,
		aesKey:      aesKey,
		metaMacXor:  metaMacXor,
		nonce:       nonce,
	}, nil
}

func (s *MegaSession) downloadData(params *MegaDownloadDataParams, dst io.Writer) error {
	block, err := aes.NewCipher(params.aesKey)
	if err != nil {
		return err
	}
	aesCtr := cipher.NewCTR(block, slices.Concat(params.nonce, make([]byte, 8)))

	req, err := http.NewRequest("GET", params.downloadUrl, nil)
	if err != nil {
		return err
	}
	resp, err := Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	io.Copy(dst, cipher.StreamReader{S: aesCtr, R: resp.Body})

	return nil
}

type FileAttr struct {
	Name string `json:"n"`
}

var attrJsonMatch = regexp.MustCompile(`{".*"}`)

// decryptAttr 解密文件属性
func decryptAttr(key []byte, ciphertext string) (attr *FileAttr, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	mode := cipher.NewCBCDecrypter(block, make([]byte, 16)) // zero IV
	data, err := base64UrlDecode(ciphertext)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, len(ciphertext))
	mode.CryptBlocks(buf, data)

	str := attrJsonMatch.FindString(string(buf))
	log.Debug("attr: ", str)
	attr = &FileAttr{}
	err = json.Unmarshal([]byte(str), attr)
	if err != nil {
		return nil, err
	}
	return
}

// unpackKey 解包节点密钥
func unpackKey(nodeKey []byte) (aesKey, metaMacXor, nonce []byte) {
	aesKey = make([]byte, 16)
	metaMacXor = make([]byte, 8)
	nonce = make([]byte, 8)
	if len(nodeKey) != 32 {
		log.Panicf("Invalid node key: (%s) %v", nodeKey, nodeKey)
	}

	put := func(b []byte, v uint32) {
		binary.LittleEndian.PutUint32(b, v)
	}
	dw := func(p []byte, n int) uint32 {
		return binary.LittleEndian.Uint32(p[n*4:])
	}

	put(aesKey[0:], dw(nodeKey, 0)^dw(nodeKey, 4))
	put(aesKey[4:], dw(nodeKey, 1)^dw(nodeKey, 5))
	put(aesKey[8:], dw(nodeKey, 2)^dw(nodeKey, 6))
	put(aesKey[12:], dw(nodeKey, 3)^dw(nodeKey, 7))

	put(nonce[0:], dw(nodeKey, 4))
	put(nonce[4:], dw(nodeKey, 5))

	put(metaMacXor[0:], dw(nodeKey, 6))
	put(metaMacXor[4:], dw(nodeKey, 7))

	return
}

func (s *MegaSession) apiRequest(req []byte) (resp []byte, err error) {
	s.apiMu.Lock()
	defer func() {
		s.sn++
		s.apiMu.Unlock()
	}()

	backOffSleep := func(t *time.Duration) {
		time.Sleep(*t)
		*t *= 2
		if *t > maxSleepTime {
			*t = maxSleepTime
		}
	}

	url := API_URL
	url += "/cs?id=" + strconv.FormatInt(s.sn, 10)
	// if s.sid != "" {
	// 	url += "&sid=" + s.sid
	// }

	var request *http.Request
	var response *http.Response
	sleepTime := minSleepTime
	for i := 0; i < RETRIES+1; i++ {
		if i != 0 {
			log.Debugf("Retry API request %d/%d: %v", i, RETRIES+1, err)
			backOffSleep(&sleepTime)
		}

		request, err = http.NewRequest("POST", url, bytes.NewReader(req))
		if err != nil {
			continue
		}
		request.Header.Set("Content-Type", "application/json")
		response, err = Client.Do(request)
		if err != nil {
			continue
		}
		if response.StatusCode != 200 {
			err = fmt.Errorf("http status: %s", response.Status)
			_ = response.Body.Close()
			continue
		}
		resp, err = io.ReadAll(response.Body)
		if err != nil {
			_ = response.Body.Close()
			continue
		}
		err = response.Body.Close()
		if err != nil {
			continue
		}
		log.Debug("API response: ", string(resp))

		if bytes.HasPrefix(resp, []byte("[")) || bytes.HasPrefix(resp, []byte("-")) {
			return nil, ErrBadResp
		}

		if len(resp) < 6 {
			var emsg [1]ErrorMsg
			err = json.Unmarshal(resp, &emsg)
			if err != nil {
				err = json.Unmarshal(resp, &emsg[0])
				if err != nil {
					return resp, ErrBadResp
				}
			}
			err = emsg[0].Parse()
			if err == ErrAgain {
				continue
			}
			return resp, err
		}

		if err == nil {
			return resp, nil
		}
	}

	return nil, err
}

type LinkType = int

const (
	LINK_NONE LinkType = iota
	LINK_FILE
	LINK_FOLDER
)

type LinkRegex struct {
	linkType LinkType
	re       *regexp.Regexp
}

var linkRegexes = []LinkRegex{
	{
		linkType: LINK_FILE,
		re: regexp.MustCompile(
			`^https?://mega(?:\.co)?\.nz/#!([a-zA-Z0-9_-]{8})!([a-zA-Z0-9_=-]{43}={0,2})$`),
	},
	{
		linkType: LINK_FILE,
		re: regexp.MustCompile(
			`^https?://mega\.nz/file/([a-zA-Z0-9_-]{8})#([a-zA-Z0-9_-]{43}={0,2})$`),
	},
	{
		linkType: LINK_FOLDER,
		re: regexp.MustCompile(
			`^https?://mega(?:\.co)?\.nz/#F!([a-zA-Z0-9_-]{8})!([a-zA-Z0-9_-]{22})(?:[!?]([a-zA-Z0-9_-]{8}))?$`),
	},
	{
		linkType: LINK_FOLDER,
		re: regexp.MustCompile(
			`^https?://mega\.nz/folder/([a-zA-Z0-9_-]{8})#([a-zA-Z0-9_-]{22})/file/([a-zA-Z0-9_-]{8})$`),
	},
	{
		linkType: LINK_FOLDER,
		re: regexp.MustCompile(
			`^https?://mega\.nz/folder/([a-zA-Z0-9_-]{8})#([a-zA-Z0-9_-]{22})/folder/([a-zA-Z0-9_-]{8})$`),
	},
	{
		linkType: LINK_FOLDER,
		re: regexp.MustCompile(
			`^https?://mega\.nz/folder/([a-zA-Z0-9_-]{8})#([a-zA-Z0-9_-]{22})$`),
	},
}

type MegaLink struct {
	Type     LinkType
	Handle   string
	Key      string
	Specific string
}

func parseLink(url string) *MegaLink {
	for _, lr := range linkRegexes {
		match := lr.re.MatchString(url)
		if match {
			matches := lr.re.FindStringSubmatch(url)
			if len(matches) > 0 {
				l := &MegaLink{
					Type:   lr.linkType,
					Handle: matches[1],
					Key:    matches[2],
				}
				if len(matches) > 3 {
					l.Specific = matches[3]
				}
				return l
			}
		}
	}
	return nil
}
