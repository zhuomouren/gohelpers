package gorequest

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var Helper = NewGoRequest()

type GoRequest struct {
	URL                      *url.URL
	Headers                  *http.Header
	HTTPClient               *http.Client
	UseCookie                bool
	requestCallbacks         []RequestCallback
	responseCallbacks        []ResponseCallback
	errorCallbacks           []ErrorCallback
	DownloadContinued        bool // 断点续传
	DownloadProgressInterval time.Duration
	downloadCallbacks        []DownloadCallback
	Logf                     AppLogFunc
	Ctx                      context.Context
	Retries                  int // 失败重试次数。如果是 -1，则一直重试，直到成功。默认是0，执行一次
	UserAgent                string
	Debug                    bool
	DumpBody                 bool
	dump                     []byte
	abort                    bool
	baseURL                  *url.URL
	// Depth is the number of the parents of the request
	Depth int
	// Method is the HTTP method of the request
	Method string
	// MaxBodySize is the limit of the retrieved response body in bytes.
	// 0 means unlimited.
	// The default value for MaxBodySize is 10MB (10 * 1024 * 1024 bytes).
	MaxBodySize int
	lock        *sync.RWMutex
}

// RequestCallback is a type alias for OnRequest callback functions
type RequestCallback func(*http.Request)

// ResponseCallback is a type alias for OnResponse callback functions
type ResponseCallback func(*GoResponse)

// ErrorCallback is a type alias for OnError callback functions
type ErrorCallback func(*GoResponse, error)

// ScrapedCallback is a type alias for OnScraped callback functions
type DownloadCallback func(int64, int64)

// 创建一个新的实例，并返回指针
func NewGoRequest(options ...func(*GoRequest)) *GoRequest {
	this := &GoRequest{}
	this.init()

	for _, f := range options {
		f(this)
	}

	return this
}

func Client(client *http.Client) func(*GoRequest) {
	return func(this *GoRequest) {
		this.HTTPClient = client
	}
}

func UserAgent(userAgent string) func(*GoRequest) {
	return func(this *GoRequest) {
		this.UserAgent = userAgent
	}
}

// 是否使用 cookie
func EnableCookie() func(*GoRequest) {
	return func(this *GoRequest) {
		jar, _ := cookiejar.New(nil)
		this.getClient().Jar = jar
	}
}
func DisableCookie() func(*GoRequest) {
	return func(this *GoRequest) {
		this.getClient().Jar = nil
	}
}

func EnableInsecureTLS(enable bool) func(*GoRequest) {
	return func(this *GoRequest) {
		trans := this.getTransport()
		if trans == nil {
			return
		}
		if trans.TLSClientConfig == nil {
			trans.TLSClientConfig = &tls.Config{}
		}
		trans.TLSClientConfig.InsecureSkipVerify = enable
	}
}

// 设置代理
func ProxyURL(rawurl string) func(*GoRequest) {
	return func(this *GoRequest) {
		trans := this.getTransport()
		if trans == nil {
			return
		}

		u, err := url.Parse(rawurl)
		if err != nil {
			this.Logf(ERROR, "parse proxy url [%s] error: %s", rawurl, err.Error())
			return
		}
		trans.Proxy = http.ProxyURL(u)
	}
}

func Context(ctx context.Context) func(*GoRequest) {
	return func(this *GoRequest) {
		this.Ctx = ctx
	}
}

// 日志
func Logf(logf AppLogFunc) func(*GoRequest) {
	return func(this *GoRequest) {
		reqLogf := func(lvl LogLevel, f string, args ...interface{}) {
			logf(LogLevel(lvl), f, args...)
		}

		this.Logf = reqLogf
	}
}

// Abort cancels the HTTP request when called in an OnRequest callback
func (this *GoRequest) Abort() {
	this.abort = true
}

// AbsoluteURL returns with the resolved absolute URL of an URL chunk.
// AbsoluteURL returns empty string if the URL chunk is a fragment or
// could not be parsed
func (this *GoRequest) AbsoluteURL(u string) string {
	if strings.HasPrefix(u, "#") {
		return ""
	}
	var base *url.URL
	if this.baseURL != nil {
		base = this.baseURL
	} else {
		base = this.URL
	}
	absURL, err := base.Parse(u)
	if err != nil {
		return ""
	}
	absURL.Fragment = ""
	if absURL.Scheme == "//" {
		absURL.Scheme = this.URL.Scheme
	}
	return absURL.String()
}

func (this *GoRequest) init() {
	if this.HTTPClient == nil {
		this.HTTPClient = newClient()
	}

	this.DownloadProgressInterval = 200 * time.Millisecond

	// log
	this.Logf = func(lvl LogLevel, f string, args ...interface{}) {
		// if lvl < ERROR {
		// 	return
		// }
		log.Println(fmt.Sprintf(lvl.String()+": "+f, args...))
	}
}

func (this *GoRequest) Fetch(method, rawurl string, requestData io.Reader, hdr http.Header, ctx context.Context, file *os.File) (*GoResponse, error) {
	parsedURL, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}

	if hdr == nil {
		hdr = http.Header{"User-Agent": []string{this.UserAgent}}
	}
	requestBody, ok := requestData.(io.ReadCloser)
	if !ok && requestData != nil {
		requestBody = ioutil.NopCloser(requestData)
	}

	// The Go HTTP API ignores "Host" in the headers, preferring the client
	// to use the Host field on Request.
	host := parsedURL.Host
	if hostHeader := hdr.Get("Host"); hostHeader != "" {
		host = hostHeader
	}

	req := &http.Request{
		Method:     method,
		Header:     make(http.Header),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Body:       requestBody,
		Host:       host,
	}
	setRequestBody(req, requestData)

	if this.Ctx != nil {
		req.WithContext(this.Ctx)
	}
	if ctx != nil {
		req.WithContext(ctx)
	}

	this.handleOnRequest(req)

	if method == "POST" && req.Header.Get("Content-Type") == "" {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}

	goResp := &GoResponse{
		request:                  req,
		downloadProgressInterval: this.DownloadProgressInterval,
		// downloadProgress:         this.downloadProgress,
	}

	if this.Debug {
		dump, err := httputil.DumpRequest(req, this.DumpBody)
		if err != nil {
			this.Logf(ERROR, err.Error())
		}

		this.dump = dump
	}

	if goResp.client == nil {
		goResp.client = this.getClient()
	}

	// 下载文件，支持断点续传
	var stat os.FileInfo
	if file != nil {
		stat, err = file.Stat()
		if err != nil {
			return nil, err
		}

		if _, ok := req.Header["Range"]; !ok && this.DownloadContinued && stat.Size() > 0 {
			req.Header.Set("Range", "bytes="+strconv.FormatInt(stat.Size(), 10)+"-")
		}
	}

	var res *http.Response
	before := time.Now()
	// retries default value is 0, it will run once.
	// retries equal to -1, it will run forever until success
	// retries is setted, it will retries fixed times.
	for i := 0; this.Retries == -1 || i <= this.Retries; i++ {
		res, err = goResp.client.Do(req)
		if err == nil {
			this.Logf(WARN, "retries %d: %s", i, err.Error())
			break
		}
	}
	after := time.Now()
	goResp.cost = after.Sub(before)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var bodyReader io.Reader = res.Body

	if file != nil || len(this.downloadCallbacks) > 0 {
		readBytes := make([]byte, 1024)
		total := res.ContentLength
		var current int64
		var lastTime time.Time

		for {
			n, err := bodyReader.Read(readBytes)
			if n > 0 {
				if file != nil {
					if _, err := file.Write(readBytes[:n]); err != nil {
						return nil, err
					}
				}
				current += int64(n)
				if now := time.Now(); now.Sub(lastTime) > this.DownloadProgressInterval {
					lastTime = now
					this.handleOnDownload(current, total)
				}
				if current >= total {
					break
				}
				if this.MaxBodySize > 0 && current >= int64(this.MaxBodySize) {
					break
				}
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
		}

		if file != nil {
			return nil, nil
		}
	} else {

	}

	// if this.MaxBodySize > 0 {
	// 	bodyReader = io.LimitReader(bodyReader, int64(this.MaxBodySize))
	// }
	contentEncoding := strings.ToLower(res.Header.Get("Content-Encoding"))
	if !res.Uncompressed && (strings.Contains(contentEncoding, "gzip") || (contentEncoding == "" && strings.Contains(strings.ToLower((res.Header.Get("Content-Type"))), "gzip"))) {
		bodyReader, err = gzip.NewReader(bodyReader)
		if err != nil {
			return nil, err
		}
		defer bodyReader.(*gzip.Reader).Close()
	}
	body, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		return nil, err
	}

	goResp.responseBody = body

	return goResp, nil
}

// OnRequest registers a function. Function will be executed on every
// request made by the Collector
func (this *GoRequest) OnRequest(f RequestCallback) {
	this.lock.Lock()
	if this.requestCallbacks == nil {
		this.requestCallbacks = make([]RequestCallback, 0, 4)
	}
	this.requestCallbacks = append(this.requestCallbacks, f)
	this.lock.Unlock()
}
func (this *GoRequest) handleOnRequest(req *http.Request) {
	if this.Debug {

	}
	for _, f := range this.requestCallbacks {
		f(req)
	}
}

// OnResponse registers a function. Function will be executed on every response
func (this *GoRequest) OnResponse(f ResponseCallback) {
	this.lock.Lock()
	if this.responseCallbacks == nil {
		this.responseCallbacks = make([]ResponseCallback, 0, 4)
	}
	this.responseCallbacks = append(this.responseCallbacks, f)
	this.lock.Unlock()
}

func (this *GoRequest) OnDownload(f DownloadCallback) {
	this.lock.Lock()
	if this.downloadCallbacks == nil {
		this.downloadCallbacks = make([]DownloadCallback, 0, 4)
	}
	this.downloadCallbacks = append(this.downloadCallbacks, f)
	this.lock.Unlock()
}
func (this *GoRequest) handleOnDownload(current, total int64) {
	if this.Debug {

	}
	for _, f := range this.downloadCallbacks {
		f(current, total)
	}
}

func (this *GoRequest) getClient() *http.Client {
	if this.HTTPClient == nil {
		this.HTTPClient = newClient()
	}

	return this.HTTPClient
}

func (this *GoRequest) getTransport() *http.Transport {
	trans, ok := this.getClient().Transport.(*http.Transport)
	if !ok {
		this.Logf(WARN, "unable to get client.Transport")
		return nil
	}
	return trans
}

// create a default client
func newClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Jar:       jar,
		Transport: transport,
		Timeout:   2 * time.Minute,
	}
}

func setRequestBody(req *http.Request, body io.Reader) {
	if body != nil {
		switch v := body.(type) {
		case *bytes.Buffer:
			req.ContentLength = int64(v.Len())
			buf := v.Bytes()
			req.GetBody = func() (io.ReadCloser, error) {
				r := bytes.NewReader(buf)
				return ioutil.NopCloser(r), nil
			}
		case *bytes.Reader:
			req.ContentLength = int64(v.Len())
			snapshot := *v
			req.GetBody = func() (io.ReadCloser, error) {
				r := snapshot
				return ioutil.NopCloser(&r), nil
			}
		case *strings.Reader:
			req.ContentLength = int64(v.Len())
			snapshot := *v
			req.GetBody = func() (io.ReadCloser, error) {
				r := snapshot
				return ioutil.NopCloser(&r), nil
			}
		}
		if req.GetBody != nil && req.ContentLength == 0 {
			req.Body = http.NoBody
			req.GetBody = func() (io.ReadCloser, error) { return http.NoBody, nil }
		}
	}
}
