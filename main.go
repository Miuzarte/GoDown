package main

import (
	"flag"
	"net/http"
	"net/url"

	log "github.com/sirupsen/logrus"
)

var (
	memUsage float64 = 1024 * 1024 * 1024 // 内存使用量, 1GB

	showTotalProgressBar  = true // 显示总进度条
	showThreadProgressBar = true // 显示线程进度条 (花里胡哨! )
)

var (
	DownloadsFolder string
	ProxyURL        *url.URL

	Header = http.Header{
		"Accept":        {"*/*"},
		"Cache-Control": {"no-cache"},
		"Connection":    {"keep-alive"},
		"User-Agent":    {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36 Edg/130.0.0.0"},
	}
	// Client 全局自定义 Client
	Client = NewClient()
)

func init() {
	log.SetFormatter(&LogFormat{})
	log.SetLevel(log.TraceLevel)
}

func Init() {
	p := flag.String("p", "", "Proxy address")
	t := flag.Int("t", 6, "Number of threads, default 6")
	bs := flag.Int("bs", 1024*1024*16, "Block size， default 16MiB")
	ll := flag.String("ll", "info", "Log level: trace, debug, info, warn/warning, error, fatal, panic")
	flag.Parse()

	if *p != "" {
		proxyURL, err := url.Parse(*p)
		if err != nil {
			log.Fatalf("Failed to parse proxy URL: %v", err)
		}
		ProxyURL = proxyURL
	}

	threadNum = *t
	blockSize = *bs

	l, err := log.ParseLevel(*ll)
	if err != nil {
		log.Fatalf("Failed to parse log level: %v", err)
	}
	log.SetLevel(l)

}

func main() {
	Init()

	args := flag.Args()
	if len(args) < 1 || args[0] == "" {
		log.Fatalf("Usage: godown <URL> [-p <proxy>] [-t <threadNum>] [-bs blockSize] [-ll <log level>] ")
	}

	j := &Job{Url: args[0]}
	j.Start()

}
