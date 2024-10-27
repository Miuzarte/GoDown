package main

import (
	"net/http"
)

type Transport struct {
	Transport http.RoundTripper
}

// RoundTrip 全局设置 Header
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// req.Header = Header // range被覆盖
	for k, v := range Header {
		req.Header[k] = v
	}

	// 设置代理
	if ProxyURL != nil {
		proxy := http.ProxyURL(ProxyURL)
		transport := t.Transport.(*http.Transport)
		transport.Proxy = proxy
	}

	return t.Transport.RoundTrip(req)
}

func NewClient() *http.Client {
	return &http.Client{
		Transport: &Transport{
			Transport: http.DefaultTransport,
		},
	}
}
