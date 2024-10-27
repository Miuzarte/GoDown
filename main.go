package main

import (
	"flag"
	"net/http"
	"net/url"

	log "github.com/sirupsen/logrus"
)

var (
	autoRetry = 3 // 每个线程的自动重试次数

	memUsage float64 = 1024 * 1024 * 1024 // 内存使用量, 1GB

	showTotalProgressBar  = true // 显示总进度条
	showThreadProgressBar = true // 显示线程进度条 (花里胡哨! )
)

var (
	DownloadsFolder string

	ProxyURL *url.URL

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

	proxy := flag.String("p", "", "Proxy address")
	logLevel := flag.String("l", "info", "Log level: trace, debug, info, warn/warning, error, fatal, panic")
	flag.Parse()

	if *proxy != "" {
		proxyURL, err := url.Parse(*proxy)
		if err != nil {
			log.Fatalf("Failed to parse proxy URL: %v", err)
		}
		ProxyURL = proxyURL
	}

	l, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatalf("Failed to parse log level: %v", err)
	}
	log.SetLevel(l)

}

func main() {
	if len(flag.Args()) < 1 {
		log.Fatalf("Usage: godown.exe <URL> [-l <log level>] [-p <proxy>]")
	}

	j := &Job{Url: flag.Arg(0)}
	j.init()
	log.Info(j)
	j.Start()

}
