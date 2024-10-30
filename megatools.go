package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	API_URL      = "https://g.api.mega.co.nz"
	RETRIES      = 10
	minSleepTime = 10 * time.Millisecond // for retries
	maxSleepTime = 5 * time.Second       // for retries
)

// MegaSession 结构体定义
type MegaSession struct {
	http            *http.Client
	maxUL           int
	maxDL           int
	proxy           string
	maxWorkers      int
	id              int64
	sid             string
	rid             string
	apiURLParams    map[string]string
	passwordSaltV2  string
	passwordKey     []byte
	passwordKeySave []byte
	masterKey       []byte
	// rsaKey			RSAKey
	userHandle string
	userName   string
	userEmail  string
	shareKeys  map[string]string
	// fsNodes		   []FSNode
	// statusCallback	StatusCallback
	statusUserdata interface{}
	lastRefresh    int64
	createPreview  bool
	resumeEnabled  bool

	// 序列化 api 请求
	apiMu sync.Mutex
}

func (ms *MegaSession) EnablePreview() {
	ms.createPreview = true
}

func makeRequestID() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	k := make([]byte, 10)
	// rand.Seed(time.Now().UnixNano())

	for i := range k {
		k[i] = chars[rand.Intn(len(chars))]
	}

	return string(k)
}

func newMegaSession() *MegaSession {
	s := &MegaSession{
		http:          NewClient(),
		id:            time.Now().Unix(),
		rid:           makeRequestID(),
		apiURLParams:  make(map[string]string),
		shareKeys:     make(map[string]string),
		resumeEnabled: true,
	}

	return s
}

type SessionFlags = int

const (
	SESSION_OPEN SessionFlags = 1 << iota
	SESSION_AUTH_ONLY
	SESSION_AUTH_OPTIONAL
)

var (
	optUsername string
	optPassword string
)

func startSession(flags SessionFlags) (*MegaSession, error) {
	// 创建一个新的 Mega 会话
	ms := newMegaSession()

	// 启用会话的预览功能
	ms.EnablePreview()

	// 如果没有设置 TOOL_SESSION_OPEN 标志，直接返回会话对象
	if flags&SESSION_OPEN == 0 {
		return ms, nil
	}

	// 允许未认证的会话
	if optUsername == "" || optPassword == "" {
		// 如果没有提供用户名或密码
		if flags&SESSION_AUTH_OPTIONAL != 0 {
			// 如果设置了 TOOL_SESSION_AUTH_OPTIONAL 标志，直接返回会话对象
			return ms, nil
		}

		// 否则，打印错误信息并返回 nil
		return nil, fmt.Errorf("ERROR: Authentication is required")
	}

	panic("unimplemented")

	/** C

	// 打开会话
	if (!mega_session_open(s, opt_username, opt_password, cache_timout, &is_new_session, &local_err)) {
		// 如果会话打开失败，打印错误信息并跳转到错误处理部分
		g_printerr("ERROR: Can't login to mega.nz: %s\n", local_err->message);
		goto err;
	}

	// 如果是新会话，保存会话
	if (is_new_session)
		mega_session_save(s, NULL);

	// 如果没有设置 TOOL_SESSION_AUTH_ONLY 标志，并且需要重新加载文件或是新会话
	if (!(flags & TOOL_SESSION_AUTH_ONLY) && (opt_reload_files || is_new_session)) {
		// 刷新会话
		if (!mega_session_refresh(s, &local_err)) {
			// 如果刷新失败，打印错误信息并跳转到错误处理部分
			g_printerr("ERROR: Can't read filesystem info from mega.nz: %s\n", local_err->message);
			goto err;
		}

		// 保存会话
		mega_session_save(s, NULL);
	}

	// 根据选项启用或禁用会话的预览功能
	mega_session_enable_previews(s, !!opt_enable_previews);
	// 根据选项设置会话的恢复功能
	mega_session_set_resume(s, !opt_disable_resume);

	// 返回会话对象
	return s;

	 **/

	// return ms, nil
}

func downloadMain(links []string) {
	// 创建会话
	s, err := startSession(SESSION_OPEN | SESSION_AUTH_ONLY | SESSION_AUTH_OPTIONAL)
	if err != nil {
		log.Fatal(err)
	}
	_ = s

	for _, link := range links {
		l := parseLink(link)
		if l == nil {
			log.Warn("Invalid link: ", link)
			continue
		}

		switch l.Type {
		case LINK_FILE:

		case LINK_FOLDER:

		default:
			log.Warn("Invalid link: ", link)
			continue
		}

	}

}

/**

enum {
	LINK_NONE,
	LINK_FILE,
	LINK_FOLDER,
};

**/

type LinkType = int

const (
	LINK_NONE LinkType = iota
	LINK_FILE
	LINK_FOLDER
)

type LinkRegex struct {
	pattern  string
	linkType LinkType
	re       *regexp.Regexp
}

var linkRegexes = []LinkRegex{
	{
		pattern:  "^https?://mega(?:\\.co)?\\.nz/#!([a-z0-9_-]{8})!([a-z0-9_=-]{43}={0,2})$",
		linkType: LINK_FILE,
		re:       regexp.MustCompile("^https?://mega(?:\\.co)?\\.nz/#!([a-z0-9_-]{8})!([a-z0-9_=-]{43}={0,2})$"),
	},
	{
		pattern:  "^https?://mega\\.nz/file/([a-z0-9_-]{8})#([a-z0-9_-]{43}={0,2})$",
		linkType: LINK_FILE,
		re:       regexp.MustCompile("^https?://mega\\.nz/file/([a-z0-9_-]{8})#([a-z0-9_-]{43}={0,2})$"),
	},
	{
		pattern:  "^https?://mega(?:\\.co)?\\.nz/#F!([a-z0-9_-]{8})!([a-z0-9_-]{22})(?:[!?]([a-z0-9_-]{8}))?$",
		linkType: LINK_FOLDER,
		re:       regexp.MustCompile("^https?://mega(?:\\.co)?\\.nz/#F!([a-z0-9_-]{8})!([a-z0-9_-]{22})(?:[!?]([a-z0-9_-]{8}))?$"),
	},
	{
		pattern:  "^https?://mega\\.nz/folder/([a-z0-9_-]{8})#([a-z0-9_-]{22})/file/([a-z0-9_-]{8})$",
		linkType: LINK_FOLDER,
		re:       regexp.MustCompile("^https?://mega\\.nz/folder/([a-z0-9_-]{8})#([a-z0-9_-]{22})/file/([a-z0-9_-]{8})$"),
	},
	{
		pattern:  "^https?://mega\\.nz/folder/([a-z0-9_-]{8})#([a-z0-9_-]{22})/folder/([a-z0-9_-]{8})$",
		linkType: LINK_FOLDER,
		re:       regexp.MustCompile("^https?://mega\\.nz/folder/([a-z0-9_-]{8})#([a-z0-9_-]{22})/folder/([a-z0-9_-]{8})$"),
	},
	{
		pattern:  "^https?://mega\\.nz/folder/([a-z0-9_-]{8})#([a-z0-9_-]{22})$",
		linkType: LINK_FOLDER,
		re:       regexp.MustCompile("^https?://mega\\.nz/folder/([a-z0-9_-]{8})#([a-z0-9_-]{22})$"),
	},
}

type MegaLink struct {
	Type     LinkType
	Handle   string
	Key      string
	Specific string
}

func parseLink(url string) *MegaLink {
	for i, lr := range linkRegexes {
		match := lr.re.MatchString(url)
		log.Debugf("%d: %v", i, match)
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
				} else {
					l.Specific = ""
				}
				return l
			}
		}
	}
	return nil
}

type MegaDownloadDataParams struct {
	nodeKey     []byte
	downloadUrl string
	nodeHandle  string
	nodeName    string
	nodeSize    uint64
}

// download 需要链接的 handle # key
func (s *MegaSession) download(handle, key string, dst io.Writer) error {
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
	// N   string `json:"n,omitempty"` // hash
}

type MegaDownloadResp [1]struct {
	G    string   `json:"g"`
	Size uint64   `json:"s"`
	Attr string   `json:"at"`
	Err  ErrorMsg `json:"e"`
}

// prepareDownload 获取 链接, 大小, 属性(文件名)
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
	size := dlResp[0].Size
	at := dlResp[0].Attr
	err = parseError(dlResp[0].Err)
	if url == "" {
		return nil, fmt.Errorf("Can't determine download url")
	}
	if size == 0 {
		return nil, fmt.Errorf("Can't determine file size")
	}
	if at == "" {
		return nil, fmt.Errorf("Can't get file attributes")
	}
	if err != nil {
		return nil, err
	}

	// 解码节点密钥
	urlKey, err := base64UrlDecode(key)
	if err != nil {
		return nil, err
	}
	if len(urlKey) != 32 {
		return nil, fmt.Errorf("Can't retrieve file key")
	}

	// 初始化解密密钥
	aesKey := make([]byte, 16)
	unpackKey(urlKey, &aesKey, nil, nil)

	fmt.Printf("at: %#v\n", at)

	// 使用解密密钥解密属性
	attr, err := decryptAttr(aesKey, at)
	if err != nil {
		return nil, err
	}

	return &MegaDownloadDataParams{
		nodeKey:     urlKey,
		downloadUrl: url,
		nodeHandle:  handle,
		nodeName:    attr.Name,
		nodeSize:    size,
	}, nil
}

type getDataState struct {
	s              *MegaSession
	ostream        io.Writer
	ctx            cipher.Block
	mac            ChunkedCbcMac
	macSaved       ChunkedCbcMac
	progressOffset uint64
	progressTotal  uint64
}

func (s *MegaSession) downloadData(params *MegaDownloadDataParams, dst io.Writer) error {
	aesKey := make([]byte, 16)
	metaMacXor := make([]byte, 8)
	nonce := make([]byte, 8)
	unpackKey(params.nodeKey, &aesKey, &nonce, &metaMacXor)

	state := getDataState{}
	mac, err := chunkedCbcMacInit8(aesKey, nonce)
	if err != nil {
		return err
	}
	state.mac = *mac

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return errors.New("Failed to init aes-ctr decryptor")
	}
	aesCtr := cipher.NewCTR(block, slices.Concat(nonce, make([]byte, 8)))

	req, err := http.NewRequest("GET", params.downloadUrl, nil)
	if err != nil {
		return err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	io.Copy(dst, cipher.StreamReader{S: aesCtr, R: resp.Body})

	return nil
}

func getChunkSize(idx int) uint {
	if idx < 8 {
		return uint((idx + 1) * 1024 * 128)
	}
	return 8 * 1024 * 128
}

type ChunkedCbcMac struct {
	ctx          cipher.Block
	chunkIdx     int
	nextBoundary uint64
	position     uint64
	chunkMacIv   []byte // 16
	chunkMac     []byte // 16
	metaMac      []byte // 16
}

func chunkedCbcMacInit(key, iv []byte) (*ChunkedCbcMac, error) {
	mac := &ChunkedCbcMac{
		chunkMacIv: make([]byte, 16),
		chunkMac:   make([]byte, 16),
		metaMac:    make([]byte, 16),
	}
	copy(mac.chunkMacIv, iv)
	copy(mac.chunkMac, mac.chunkMacIv)
	mac.nextBoundary = uint64(getChunkSize(mac.chunkIdx))

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	mac.ctx = block

	return mac, nil
}

func chunkedCbcMacInit8(key, iv []byte) (*ChunkedCbcMac, error) {
	return chunkedCbcMacInit(key, slices.Repeat(iv, 2))
}

func a32_to_bytes(a []uint32) ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.Grow(len(a) * 4)
	for _, v := range a {
		err := binary.Write(buf, binary.BigEndian, v)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

type FileAttr struct {
	Name string `json:"n"`
}

var attrMatch = regexp.MustCompile(`{".*"}`)

func decryptAttr(key []byte, data string) (attr FileAttr, err error) {
	err = ErrBadAttr
	block, err := aes.NewCipher(key)
	if err != nil {
		return attr, err
	}
	iv, err := a32_to_bytes([]uint32{0, 0, 0, 0})
	if err != nil {
		return attr, err
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	buf := make([]byte, len(data))
	ddata, err := base64UrlDecode(data)

	fmt.Printf("decoded: %s\n", ddata)
	fmt.Printf("len(ddata): %v\n", len(ddata))

	if err != nil {
		return attr, err
	}
	mode.CryptBlocks(buf, ddata)

	fmt.Printf("decrypted: %s\n", buf)
	fmt.Printf("len(buf): %v\n", len(buf))

	if string(buf[:4]) == "MEGA" {
		str := strings.TrimRight(string(buf[4:]), "\x00")
		trimmed := attrMatch.FindString(str)
		if trimmed != "" {
			str = trimmed
		}
		err = json.Unmarshal([]byte(str), &attr)
	}
	return attr, err
}

func dw(p []byte, n int) uint32 {
	return binary.LittleEndian.Uint32(p[n*4:])
}

func put(b []byte, v uint32) {
	binary.LittleEndian.PutUint32(b, v)
}

func unpackKey(nodeKey []byte, aesKey, nonce, metaMacXor *[]byte) {
	if len(nodeKey) != 32 {
		log.Panicf("Invalid node key: (%s) %v", nodeKey, nodeKey)
	}

	if aesKey != nil {
		put((*aesKey)[0:], dw(nodeKey, 0)^dw(nodeKey, 4))
		put((*aesKey)[4:], dw(nodeKey, 1)^dw(nodeKey, 5))
		put((*aesKey)[8:], dw(nodeKey, 2)^dw(nodeKey, 6))
		put((*aesKey)[12:], dw(nodeKey, 3)^dw(nodeKey, 7))
	}

	if nonce != nil {
		put((*nonce)[0:], dw(nodeKey, 4))
		put((*nonce)[4:], dw(nodeKey, 5))
	}

	if metaMacXor != nil {
		put((*metaMacXor)[0:], dw(nodeKey, 6))
		put((*metaMacXor)[4:], dw(nodeKey, 7))
	}
}

func base64UrlDecode(str string) ([]byte, error) {
	if str == "" {
		return nil, errors.New("input string is nil")
	}

	str = strings.ReplaceAll(str, "-", "+")
	str = strings.ReplaceAll(str, "_", "/")

	eqs := (len(str) * 3) & 0x03
	for i := 0; i < eqs; i++ {
		str += "="
	}

	return base64.StdEncoding.DecodeString(str)
}

func base64urldecode(s string) ([]byte, error) {
	enc := base64.RawURLEncoding
	// mega base64 decoder accepts the characters from both URLEncoding and StdEncoding
	// though nearly all strings are URL encoded
	s = strings.Replace(s, "+", "-", -1)
	s = strings.Replace(s, "/", "_", -1)
	return enc.DecodeString(s)
}

func (s *MegaSession) apiRequest(req []byte) (resp []byte, err error) {
	s.apiMu.Lock()
	defer func() {
		s.id++
		s.apiMu.Unlock()
	}()

	url := API_URL
	url += "/cs?id=" + strconv.FormatInt(s.id, 10)
	if s.sid != "" {
		url += "&sid=" + s.sid
	}

	var response *http.Response
	sleepTime := minSleepTime
	for i := 0; i < RETRIES+1; i++ {
		if i != 0 {
			log.Debugf("Retry API request %d/%d: %v", i, RETRIES+1, err)
			backOffSleep(&sleepTime)
		}

		response, err = Client.Post(url, "application/json", bytes.NewReader(req))
		if err != nil {
			continue
		}
		if response.StatusCode != 200 {
			err = errors.New("Http Status: " + response.Status)
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

		if !responseOk(resp) {
			return nil, ErrBadResp
		}

		if len(resp) < 6 {
			var emsg [1]ErrorMsg
			err = json.Unmarshal(resp, &emsg)
			if err != nil {
				err = json.Unmarshal(resp, &emsg[0])
			}
			if err != nil {
				return resp, ErrBadResp
			}
			err = parseError(emsg[0])
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

func responseOk(resp []byte) bool {
	return bytes.HasPrefix(resp, []byte("[")) || bytes.HasPrefix(resp, []byte("-"))
}

func backOffSleep(t *time.Duration) {
	time.Sleep(*t)
	*t *= 2
	if *t > maxSleepTime {
		*t = maxSleepTime
	}
}

// isDirectory 检查文件是否为目录
func isDirectory(file *os.File) bool {
	fileInfo, err := file.Stat()
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}
