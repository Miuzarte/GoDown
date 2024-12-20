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

func ExportMegaLink(link string) (decryptMw func(io.Reader) io.Reader, err error) {
	s := NewMegaSession()

	l := parseLink(link)
	if l == nil {
		return nil, fmt.Errorf("invalid link: %s", link)
	}

	switch l.Type {
	case LINK_FILE:
		params, err := s.prepareDownload(l.Handle, l.Key)
		if err != nil {
			return nil, err
		}
		return params.Export()

	case LINK_FOLDER:
		panic("Not implemented")
		// return s.Download(dst, l.Handle, l.Key, l.Specific)
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
	apiURLParams map[string]string
	// passwordSaltV2 string
	// passwordKey []byte
	// passwordKeySave []byte
	masterKey []byte
	// rsaKey RSAKey
	// userHandle string
	// userName string
	// userEmail string
	shareKeys map[string]string
	// fsNodes []FSNode
	// statusCallback
	// statusUserdata interface{}
	// lastRefresh int64
	// createPreview bool
	// resumeEnabled bool
	skMap map[string]string

	apiMu sync.Mutex // 序列化 api 请求
}

func NewMegaSession() *MegaSession {
	return &MegaSession{
		sn:           time.Now().Unix(),
		apiURLParams: make(map[string]string),
	}
}

type MegaDownloadDataParams struct {
	downloadUrl string
	nodeName    string
	nodeSize    uint64
	aesKey      []byte
	nonce       []byte
	// metaMacXor  []byte // 计算文件 MAC 用, 不实现
}

// Export 导出解密中间件
func (p *MegaDownloadDataParams) Export() (decryptMw func(io.Reader) io.Reader, err error) {
	block, err := aes.NewCipher(p.aesKey)
	if err != nil {
		return nil, err
	}
	return func(r io.Reader) io.Reader {
		return cipher.StreamReader{
			S: cipher.NewCTR(
				block,
				slices.Concat(p.nonce, make([]byte, 8)),
			),
			R: r,
		}
	}, nil
}

type MegaDownloadReq [1]struct {
	Cmd string `json:"a"`
	G   int    `json:"g"`
	SSL int    `json:"ssl,omitempty"`
	P   string `json:"p,omitempty"` // handle for file share
	N   string `json:"n,omitempty"` // hash for folder share
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
	// if strings.Contains(key, ":") {
	// 	key = strings.Split(key, ":")[1]
	// }
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
		nonce:       nonce,
		// metaMacXor:  metaMacXor,
	}, nil
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
	for k, v := range s.apiURLParams {
		url += "&" + k + "=" + v
	}

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

		if !bytes.HasPrefix(resp, []byte("[")) && !bytes.HasPrefix(resp, []byte("-")) {
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

type FilesMsg [1]struct {
	Cmd string `json:"a"`
	C   int    `json:"c"`
}

const (
	MEGA_NODE_FILE    = 0
	MEGA_NODE_FOLDER  = 1
	MEGA_NODE_ROOT    = 2
	MEGA_NODE_INBOX   = 3
	MEGA_NODE_TRASH   = 4
	MEGA_NODE_NETWORK = 9
	MEGA_NODE_CONTACT = 8
)

type FSNode struct {
	Hash   string `json:"h"`
	Parent string `json:"p"`
	User   string `json:"u"`
	T      int    `json:"t"`
	Attr   string `json:"a"`
	Key    string `json:"k"`
	Ts     int64  `json:"ts"`
	SUser  string `json:"su"`
	SKey   string `json:"sk"`
	Size   int64  `json:"s"`
}

type FilesResp [1]struct {
	F []FSNode `json:"f"`

	Ok []struct {
		Hash string `json:"h"`
		Key  string `json:"k"`
	} `json:"ok"`

	S []struct {
		Hash string `json:"h"`
		User string `json:"u"`
	} `json:"s"`
	User []struct {
		User  string `json:"u"`
		C     int    `json:"c"`
		Email string `json:"m"`
	} `json:"u"`
	Sn string `json:"sn"`
}

func (s *MegaSession) OpenFolder(handle, key, specific string) ([]FSNode, error) {

	return nil, nil
}

type NodeMeta struct {
	key     []byte
	compkey []byte
	iv      []byte
	mac     []byte
}

// Mega filesystem object
type MegaFS struct {
	root   *Node
	trash  *Node
	inbox  *Node
	sroots []*Node
	lookup map[string]*Node
	skmap  map[string]string
	mutex  sync.Mutex
}

// Filesystem node
type Node struct {
	fs       *MegaFS
	name     string
	hash     string
	parent   *Node
	children []*Node
	ntype    int
	size     int64
	ts       time.Time
	meta     NodeMeta
}

func bytes2u32s(b []byte) ([]uint32, error) {
	length := len(b) + 3
	a := make([]uint32, length/4)
	buf := bytes.NewBuffer(b)
	for i := range a {
		err := binary.Read(buf, binary.BigEndian, &a[i])
		if err != nil {
			return nil, err
		}
	}

	return a, nil
}

func (s *MegaSession) parseFSNode(item FSNode) (*MegaDownloadDataParams, error) {

	return nil, nil
}
