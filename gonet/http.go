package gonet

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

type GoHTTP struct{}

var HTTPHelper = &GoHTTP{}

func (this *GoHTTP) GET(rawurl, proxy string, headerMap map[string]string, timeout int64) ([]byte, error) {
	if timeout <= 0 {
		timeout = 30
	}

	_timeout := time.Duration(timeout) * time.Second

	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   _timeout,
			KeepAlive: _timeout,
		}).Dial,
		DisableKeepAlives: true,
		// 建立TLS连接的时候，不验证服务器的证书
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		TLSHandshakeTimeout: _timeout,
	}

	// 设置代理
	if proxy != "" {
		pu, err := url.Parse(proxy)
		if err != nil {
			return nil, err
		}

		transport.Proxy = http.ProxyURL(pu)
	}

	client := &http.Client{
		Timeout:   _timeout,
		Transport: transport,
	}

	req, err := http.NewRequest("GET", rawurl, nil)
	if err != nil {
		return nil, err
	}
	req.Close = true

	for name, value := range headerMap {
		if name != "" {
			req.Header.Set(name, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}
