package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vbauerster/mpb/v8"
)

const (
	BS = 1024 * 1024 * 16
)

var (
	threadNum = 6                // 线程数
	blockSize = 1024 * 1024 * 16 // 动态多线程块大小
)

type Job struct {
	Url      string
	fileName string
	filePath string
	size     int

	ctx      context.Context
	cancel   context.CancelFunc
	progress *mpb.Progress
	fs       *os.File
	Blocks   Blocks

	// 放结构体里显示顺序全乱, 疑难杂症
	// totalBar   *mpb.Bar
	// writingBar *mpb.Bar

}

type Blocks []*Block

type Block struct {
	start   int
	end     int
	Done    chan bool // 同步顺序写入的信号
	Written int64     // 已写入硬盘的字节数
	bytes.Buffer
}

func (j Job) String() string {
	return fmt.Sprintf("fileName: %s, size: %s, Url: %s", j.fileName, FormatBytes(j.size), j.Url)
}

func (j *Job) init() error {
	j.ctx, j.cancel = context.WithCancel(context.Background())
	j.progress = j.newProgressWithCtx()

	if j.size > 0 {
		return nil
	}

	err := j.fetchHeader()
	if err != nil {
		return err
	}

	j.splitBlocks()

	return j.createFile()
}

// fetchHeader 获取文件头信息
func (j *Job) fetchHeader() error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second*5, // 5s 拿 header
	)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", j.Url, nil)
	if err != nil {
		return err
	}

	resp, err := Client.Do(req)
	switch err {
	case context.DeadlineExceeded:
		return fmt.Errorf("header request timeout")
	// case context.Canceled:
	case nil:
		break
	default:
		return err
	}
	defer resp.Body.Close()

	// fmt.Println("headers:")
	// for k, v := range resp.Header {
	// 	fmt.Printf("k: %v, v: %v\n", k, v)
	// }

	j.size = int(resp.ContentLength)
	if j.size <= 0 {
		return fmt.Errorf("failed to get file size")
	}
	j.fileName = resp.Header.Get("Content-Disposition")
	if j.fileName == "" { // 取 URL 最后一段
		j.fileName = path.Base(resp.Request.URL.Path)
	}

	return nil
}

// splitBlocks 初始化块信息
func (j *Job) splitBlocks() {
	numBlocks := (j.size + blockSize - 1) / blockSize
	if numBlocks < 1 {
		numBlocks = 1
	}
	j.Blocks = make(Blocks, numBlocks)
	for i := 0; i < numBlocks; i++ {
		start := blockSize * i     // 左闭
		end := blockSize*(i+1) - 1 // 右闭
		if i == numBlocks-1 {
			end = j.size - 1
		}
		j.Blocks[i] = &Block{
			start: start,
			end:   end,
		}
	}
}

// createFile 创建文件
func (j *Job) createFile() error {
	if j.fs != nil {
		return nil
	}

	j.filePath = GetUniqueFilePath(filepath.Join(DownloadsFolder, j.fileName))
	fs, err := os.Create(j.filePath)
	if err != nil {
		return err
	}
	j.fs = fs
	return nil
}

// setupChannels 初始化块信号
func (j *Job) setupChannels() {
	for _, block := range j.Blocks {
		if block.Len() == 0 && block.Written == 0 { // 新的, 未完成的块
			block.Done = make(chan bool, 1)
		}
	}
}

var sigChan = make(chan os.Signal, 1)

func (j *Job) Start() {
S:
	err := j.init()
	if err != nil {
		log.Fatalf("Failed to init job: %v", err)
	}
	j.setupChannels()
	defer j.Clean()

	go func() {
		signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
		for {
			select {
			case <-sigChan:
				// 外部取消
				j.cancel()
				signal.Stop(sigChan)
				close(sigChan)
				return
			case <-j.ctx.Done():
				// 等待重试
				continue
			}
		}
	}()

	wg := &sync.WaitGroup{}
	wg.Add(len(j.Blocks))
	go func() {
		err := j.MergeIntoFileSyncSeq(wg)
		switch err {
		case nil:
		case context.Canceled:
		default:
			j.cancel()
		}
	}()
	err = j.DownloadIntoRam()
	if err != nil && err != context.Canceled && strings.Contains(err.Error(), "context canceled") {
		err = context.Canceled
	}
	switch err {
	case nil:
		wg.Wait()
		<-time.After(time.Millisecond * 400) // 等待进度条移除
	case context.Canceled:
		log.Warn("Download canceled")

	default:
		j.cancel()

		log.Errorf("Download failed: %v\n", err)
		fmt.Print("Retry? (y/n): ")
		var input string
		fmt.Scanln(&input)
		if strings.TrimSpace(strings.ToLower(input)) == "y" {
			goto S
		}
	}

}

func (j *Job) Clean() {
	j.fs.Close()

	fileInfo, err := os.Stat(j.filePath)
	if err != nil {
		log.Fatalf("Failed to get file info: %v", err)
	}
	if fileInfo.Size() != int64(j.size) { // 未完成下载
		os.Remove(j.filePath)
	} else { // 打印路径
		log.Infof("Downloaded file: %s", Hyperlink(j.filePath))
	}
}

// DownloadIntoRam 下载到内存
func (j *Job) DownloadIntoRam() error {
	startTime := time.Now()
	var totalBar *mpb.Bar
	if showTotalProgressBar {
		totalBar = j.newTotalBar()
	}

	wg := &sync.WaitGroup{}
	limiter := NewLimiter(threadNum)
	errChan := make(chan error, len(j.Blocks))
	for _, block := range j.Blocks {
		if block.Len() != 0 { // 未完成的块一定为 0
			continue
		}

		wg.Add(1)
		limiter.Acquire()
		go func(block *Block) {
			defer func() {
				wg.Done()
				limiter.Release()
			}()
			var err error
			// 协程内重试, 不解除槽位占用
			for i := 0; i < autoRetry; i++ {
				err = j.downloadBlock(block)
				switch err {
				case nil: // 成功, 报告 Done 后释放
					block.Done <- true
					if showTotalProgressBar {
						totalBar.EwmaIncrement(time.Since(startTime))
					}
					return
				case context.Canceled: // 直接返回 canceled error
					errChan <- err
					return
				default:
				}
				<-time.After(time.Second * time.Duration(1+i)) // 重试间隔
			}
			// 失败 autoRetry 次, 报告 Done, err 后释放
			block.Done <- false
			errChan <- err
		}(block)

	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

// downloadBlock 下载块
func (j *Job) downloadBlock(block *Block) error {
	req, err := http.NewRequestWithContext(j.ctx, "GET", j.Url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", block.start, block.end))

	resp, err := Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var src io.ReadCloser
	if showThreadProgressBar {
		src = j.newThreadBar(block).ProxyReader(resp.Body)
	} else {
		src = resp.Body
	}
	_, err = io.Copy(block, src)
	if err != nil {
		block.Reset() // 保证未完成的块一定为 0
		return err
	}

	return nil
}

// MergeIntoFile 一次性合并到文件
func (j *Job) MergeIntoFile() error {
	var err error
	for _, block := range j.Blocks {
		block.Written, err = io.Copy(j.fs, block)
		if err != nil {
			return err
		}
	}
	return nil
}

// MergeIntoFileSyncSeq 同步顺序写入到文件
func (j *Job) MergeIntoFileSyncSeq(wg *sync.WaitGroup) error {
	var dst io.WriteCloser
	if showTotalProgressBar {
		writingBar := j.newWritingBar()
		dst = writingBar.ProxyWriter(j.fs)
	} else {
		dst = j.fs
	}

	var err error
	for i, block := range j.Blocks {
		if block.Written > 0 { // 已写入
			continue
		}

		select {
		case <-j.ctx.Done():
			return j.ctx.Err()

		case done := <-block.Done:
			if !done {
				return fmt.Errorf("block %d download failed", i)
			}
			block.Written, err = io.Copy(dst, block)
			if err != nil {
				return err
			}
			block.Reset() // 释放内存
			wg.Done()

		}
		time.Sleep(time.Millisecond * 100) // slow down
	}
	return nil
}