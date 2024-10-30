package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	MegaLinkFile = [...]string{
		// https://mega.nz/#!<文件ID>!<文件密钥>
		// https://mega.co.nz/#!<文件ID>!<文件密钥>
		"^https?://mega(?:\\.co)?\\.nz/#!([a-z0-9_-]{8})!([a-z0-9_=-]{43}={0,2})$",
		// https://mega.nz/file/<文件ID>#<文件密钥>
		"^https?://mega\\.nz/file/([a-z0-9_-]{8})#([a-z0-9_-]{43}={0,2})$",
	}
	MegaLinkFolder = [...]string{
		// https://mega.nz/#F!<文件夹ID>!<文件夹密钥>
		// https://mega.co.nz/#F!<文件夹ID>!<文件夹密钥>
		"^https?://mega(?:\\.co)?\\.nz/#F!([a-z0-9_-]{8})!([a-z0-9_-]{22})(?:[!?]([a-z0-9_-]{8}))?$",
		// https://mega.nz/folder/<文件夹ID>#<文件夹密钥>/file/<文件ID>
		"^https?://mega\\.nz/folder/([a-z0-9_-]{8})#([a-z0-9_-]{22})/file/([a-z0-9_-]{8})$",
		// https://mega.nz/folder/<文件夹ID>#<文件夹密钥>/folder/<子文件夹ID>
		"^https?://mega\\.nz/folder/([a-z0-9_-]{8})#([a-z0-9_-]{22})/folder/([a-z0-9_-]{8})$",
		// https://mega.nz/folder/<文件夹ID>#<文件夹密钥>
		"^https?://mega\\.nz/folder/([a-z0-9_-]{8})#([a-z0-9_-]{22})$",
	}
)

type apiRequest struct {
	ID int    `json:"id"`
	A  string `json:"a"`
	G  int    `json:"g,omitempty"`
	P  string `json:"p,omitempty"`
	N  string `json:"n,omitempty"`
}

type apiResponse struct {
	G   string `json:"g,omitempty"`  // 下载 URL
	S   int64  `json:"s,omitempty"`  // 文件大小
	At  string `json:"at,omitempty"` // 文件属性，加密的
	Err int    `json:"e,omitempty"`  // 错误代码
}

func (j *Job) fetchMega() error {

	return nil
}

func (j *Job) initDecryptor() error {

	return nil
}

func (j *Job) decryptCTR(key []byte, in io.ReadCloser, out io.WriteCloser) error {
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(in, iv); err != nil {
		return err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	stream := cipher.NewCTR(block, iv)
	reader := &cipher.StreamReader{S: stream, R: in}

	if _, err := io.Copy(out, reader); err != nil {
		return err
	}

	return nil
}

func getFileInfo(fileId string) (downloadURL string, size int64, at string, err error) {
	const MegaApi = "https://g.api.mega.co.nz/cs?id=1"

	reqBody, err := json.Marshal([]apiRequest{
		{
			ID: 0,
			A:  "g",
			G:  1,
			P:  fileId,
		},
	})
	if err != nil {
		return "", 0, "", err
	}

	resp, err := http.Post(MegaApi, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return "", 0, "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, "", err
	}

	var apiResponses []apiResponse
	err = json.Unmarshal(respBody, &apiResponses)
	if err != nil {
		return "", 0, "", err
	}

	if len(apiResponses) == 0 {
		return "", 0, "", fmt.Errorf("API 响应为空")
	}

	if apiResponses[0].Err != 0 {
		return "", 0, "", fmt.Errorf("API 返回错误代码: %d", apiResponses[0].Err)
	}

	downloadURL = apiResponses[0].G
	size = apiResponses[0].S
	at = apiResponses[0].At

	return downloadURL, size, at, nil
}

// decryptAES128ECB 解密文件属性
func decryptAES128ECB(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext is not a multiple of the block size")
	}

	plaintext := make([]byte, len(ciphertext))
	for bs, be := 0, block.BlockSize(); bs < len(ciphertext); bs, be = bs+block.BlockSize(), be+block.BlockSize() {
		block.Decrypt(plaintext[bs:be], ciphertext[bs:be])
	}

	return plaintext, nil
}

func urlB64ToStdB64(urlB64 string) string {
	urlB64 = strings.ReplaceAll(urlB64, "-", "+")
	urlB64 = strings.ReplaceAll(urlB64, "_", "/")
	r := len(urlB64) * 6 % 8
	for r != 0 {
		urlB64 += "="
		r = (r + 6) % 8
	}
	return urlB64
}
