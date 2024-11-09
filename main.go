package main

import (
	"flag"
	"net/http"
	"net/url"
	"os"

	log "github.com/sirupsen/logrus"
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

var (
	showTotalProgressBar  bool // 显示总进度条
	showThreadProgressBar bool // 显示线程进度条 (花里胡哨! )
)

func init() {
	log.SetFormatter(&LogFormat{})
	log.SetLevel(log.TraceLevel)
}

func Init() {
	dir := flag.String("d", "", "Download directory")
	p := flag.String("p", "", "Proxy address")
	t := flag.Int("t", 6, "Number of threads")
	bs := flag.Int("bs", 1024*1024*16, "Block size")
	ll := flag.String("ll", "info", "Log level: trace, debug, info, warn/warning, error, fatal, panic")
	pbt := flag.Bool("pbt", true, "Show total progress bar")
	pbs := flag.Bool("pbs", true, "Show thread progress bar")
	flag.Parse()

	if *dir != "" {
		DownloadsFolder = *dir
	} else {
		if DownloadsFolder = GetDownloadsFolder(); DownloadsFolder == "" {
			log.Fatalf("Failed to get downloads folder, please set it manually")
		}
	}

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

	showTotalProgressBar = *pbt
	showThreadProgressBar = *pbs
}

func main() {
	Init()

	args := flag.Args()
	if len(args) < 1 || args[0] == "" {
		flag.Usage()
		os.Exit(1)
	}

	j := &Job{Url: args[0]}
	j.Start()

}
