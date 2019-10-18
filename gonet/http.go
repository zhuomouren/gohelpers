package gonet

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/net/html/charset"
)

// var DefaultUserAgent string = "abcdefghijklmnopqrstuvwxyz"
var DefaultUserAgent string = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3865.90 Safari/537.36"

var HTTPRequestHelper = NewRequest()

type Request struct {
	headers                  http.Header
	defaultClient            *http.Client
	client                   *http.Client
	cookie                   *http.Cookie
	useCookie                bool
	defaultCookieJar         http.CookieJar
	proxy                    func(*http.Request) (*url.URL, error)
	insecureTLSSkipVerify    bool
	requestCallbacks         []RequestCallback
	downloadProgressInterval time.Duration
	downloadCallbacks        []DownloadCallback
	logf                     LogFunc
	ctx                      context.Context
	retries                  int // 失败重试次数。如果是 -1，则一直重试，直到成功。默认是0，执行一次
	userAgent                string
	debug                    bool
	dump                     []byte
	// MaxBodySize is the limit of the retrieved response body in bytes.
	// 0 means unlimited.
	// The default value for MaxBodySize is 10MB (10 * 1024 * 1024 bytes).
	maxBodySize       int
	lock              *sync.RWMutex
	statusCode        int
	cost              time.Duration
	data              []byte
	err               error
	response          *http.Response
	characterEncoding string
	connectTimeout    time.Duration
	readWriteTimeout  time.Duration
	clientTimeout     time.Duration
}

// RequestCallback is a type alias for OnRequest callback functions
type RequestCallback func(*http.Request)

// DownloadCallback is a type alias for OnDownload callback functions
type DownloadCallback func(int64, int64)

// 创建一个新的实例，并返回指针
func NewRequest() *Request {
	this := &Request{}
	this.init()

	return this
}

func (this *Request) Bytes() ([]byte, error) {
	if this.err != nil {
		return nil, this.err
	}

	if this.data != nil {
		return this.data, nil
	}

	defer this.response.Body.Close()

	data, err := ioutil.ReadAll(this.response.Body)
	if err != nil {
		this.err = err
		return nil, err
	}
	this.data = data

	return this.data, nil
}

func (this *Request) String() (string, error) {
	_, err := this.Bytes()
	if err != nil {
		return "", err
	}

	data := string(this.data)
	return this.fixCharset(data)
}

func (this *Request) JSON(v interface{}) error {
	_, err := this.Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(this.data, v)
}

func (this *Request) XML(v interface{}) error {
	_, err := this.Bytes()
	if err != nil {
		return err
	}

	return xml.Unmarshal(this.data, v)
}

func (this *Request) Save(fileName string) error {
	if this.err != nil {
		return this.err
	}

	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	if this.data != nil {
		_, err = f.Write(this.data)
		return err
	}

	if len(this.downloadCallbacks) > 0 {
		if err := this.download(f); err != nil {
			return err
		}

		return nil
	}

	defer this.response.Body.Close()
	_, err = io.Copy(f, this.response.Body)

	return ioutil.WriteFile(fileName, this.data, 0644)
}

func (this *Request) SetContentType(contentType string) *Request {
	this.headers.Set("Content-Type", contentType)
	return this
}

func (this *Request) Response() *http.Response {
	return this.response
}

func (this *Request) GET(URL string) *Request {
	this.err = this.Fetch("GET", URL, nil, nil, nil)
	return this
}

// req.POST("http://example.com/login", map[string]string{"username": "admin", "password": "admin"})
func (this *Request) POST(URL string, requestData map[string]string) *Request {
	this.err = this.Fetch("POST", URL, createFormReader(requestData), nil, nil)
	return this
}

// payload := []byte(`{"user":{"email":"anon@example.com","password":"mypassword"}}`)
// req.POSTRaw("http://example.com/login", payload)
func (this *Request) POSTRaw(URL string, requestData []byte) *Request {
	this.err = this.Fetch("POST", URL, bytes.NewReader(requestData), nil, nil)
	return this
}

func (this *Request) SendJSON(URL string, requestData map[string]string) *Request {
	this.headers.Set("Content-Type", "application/json; charset=UTF-8")
	data, err := json.Marshal(requestData)
	if err != nil {
		this.err = err
		return this
	}
	this.err = this.Fetch("POST", URL, bytes.NewReader(data), nil, nil)
	return this
}

// func generateFormData() map[string][]byte {
// 	f, _ := os.Open("gocolly.jpg")
// 	defer f.Close()

// 	imgData, _ := ioutil.ReadAll(f)

// 	return map[string][]byte{
// 		"firstname": []byte("one"),
// 		"lastname":  []byte("two"),
// 		"email":     []byte("onetwo@example.com"),
// 		"file":      imgData,
// 	}
// }
// req.POSTMultipart("http://localhost:8080/", generateFormData())
func (this *Request) POSTMultipart(URL string, requestData map[string][]byte) *Request {
	boundary := randomBoundary()
	this.headers.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	this.err = this.Fetch("POST", URL, createMultipartReader(boundary, requestData), nil, nil)
	return this
}

func (this *Request) SetClient(client *http.Client) *Request {
	this.client = client
	return this
}

func (this *Request) SetUserAgent(userAgent string) *Request {
	this.userAgent = userAgent
	return this
}

func (this *Request) SetTimeout(timeout time.Duration) *Request {
	this.clientTimeout = timeout
	return this
}
func (this *Request) SetConnectTimeout(connectTimeout time.Duration) *Request {
	this.connectTimeout = connectTimeout
	return this
}
func (this *Request) SetReadWriteTimeout(readWriteTimeout time.Duration) *Request {
	this.readWriteTimeout = readWriteTimeout
	return this
}

func (this *Request) SetCharacterEncoding(characterEncoding string) *Request {
	this.characterEncoding = characterEncoding
	return this
}

func (this *Request) SetRetries(retries int) *Request {
	this.retries = retries
	return this
}

// 是否使用 cookie
func (this *Request) EnableCookie() *Request {
	return this.UseCookie(true)
}
func (this *Request) DisableCookie() *Request {
	return this.UseCookie(false)
}
func (this *Request) UseCookie(use bool) *Request {
	this.useCookie = use
	return this
}
func (this *Request) SetCookie(cookie *http.Cookie) *Request {
	this.cookie = cookie
	return this
}

func (this *Request) EnableInsecureTLSSkipVerify() *Request {
	return this.SetInsecureTLSSkipVerify(true)
}
func (this *Request) DisableInsecureTLSSkipVerify() *Request {
	return this.SetInsecureTLSSkipVerify(false)
}
func (this *Request) SetInsecureTLSSkipVerify(skip bool) *Request {
	this.insecureTLSSkipVerify = skip
	return this
}

// debug
func (this *Request) EnableDebug() *Request {
	return this.Debug(true)
}
func (this *Request) DisableDebug() *Request {
	return this.Debug(false)
}
func (this *Request) Debug(debug bool) *Request {
	this.debug = debug
	return this
}

func (this *Request) Dump() []byte {
	return this.dump
}

// 设置代理
func (this *Request) SetProxyURL(proxyURL string) *Request {
	u, err := url.Parse(proxyURL)
	if err != nil {
		this.logf(ERROR, "parse proxy url [%s] error: %s", proxyURL, err.Error())
		return this
	}

	return this.SetProxy(u)
}

func (this *Request) SetProxy(proxyURL *url.URL) *Request {
	this.proxy = http.ProxyURL(proxyURL)
	return this
}
func (this *Request) SetProxyFunc(proxy func(*http.Request) (*url.URL, error)) *Request {
	this.proxy = proxy
	return this
}

func (this *Request) AddHeader(key, value string) *Request {
	this.headers.Add(key, value)
	return this
}
func (this *Request) SetHeader(headers http.Header) *Request {
	this.headers = headers
	return this
}

func (this *Request) SetContext(ctx context.Context) *Request {
	this.ctx = ctx
	return this
}

// 日志
func (this *Request) Logf(logf LogFunc) *Request {
	reqLogf := func(lvl LogLevel, f string, args ...interface{}) {
		logf(LogLevel(lvl), f, args...)
	}

	this.logf = reqLogf
	return this
}

func (this *Request) Error() error {
	return this.err
}

func (this *Request) init() {
	this.useCookie = true
	this.insecureTLSSkipVerify = true
	this.userAgent = DefaultUserAgent
	this.headers = http.Header{"User-Agent": []string{this.userAgent}}
	this.downloadProgressInterval = 200 * time.Millisecond
	this.connectTimeout = 30 * time.Second
	this.readWriteTimeout = 30 * time.Second
	this.cost = time.Duration(0)
	this.clientTimeout = 2 * time.Minute

	// log
	this.logf = func(lvl LogLevel, f string, args ...interface{}) {
		// if lvl < ERROR {
		// 	return
		// }
		log.Println(fmt.Sprintf(lvl.String()+" "+f, args...))
	}
}

func (this *Request) Fetch(method, rawurl string, requestData io.Reader, hdr http.Header, ctx context.Context) error {
	this.data, this.err, this.response, this.dump = nil, nil, nil, nil
	this.cost = time.Duration(0)

	parsedURL, err := url.Parse(rawurl)
	if err != nil {
		this.err = err
		return err
	}

	if hdr == nil {
		hdr = this.headers
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
		URL:        parsedURL,
		Header:     hdr,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Body:       requestBody,
		Host:       host,
	}
	setRequestBody(req, requestData)

	if this.ctx != nil {
		req.WithContext(this.ctx)
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

	if this.debug {
		dump, err := httputil.DumpRequest(req, true)
		if err != nil {
			this.err = err
			this.logf(ERROR, err.Error())
		}

		this.logf(DEBUG, "Response:\n%s", string(dump))
		this.dump = dump
	}

	var resp *http.Response
	before := time.Now()
	// retries default value is 0, it will run once.
	// retries equal to -1, it will run forever until success
	// retries is setted, it will retries fixed times.
	for i := 0; this.retries == -1 || i <= this.retries; i++ {
		resp, err = this.getClient().Do(req)
		if err != nil {
			this.logf(WARN, "retries %d: %s", i, err.Error())
		} else {
			break
		}
	}
	after := time.Now()
	this.cost = after.Sub(before)
	if err != nil {
		this.err = err
		return err
	}

	this.statusCode = resp.StatusCode
	if this.debug {
		dump, err := httputil.DumpResponse(resp, false)
		if err != nil {
			this.err = err
			this.logf(ERROR, err.Error())
		}

		this.logf(DEBUG, "Response:\n%s", string(dump))
		this.dump = bytes.Join([][]byte{this.dump, dump}, []byte("\n"))
	}

	var bodyReader io.Reader = resp.Body
	if this.maxBodySize > 0 {
		bodyReader = io.LimitReader(bodyReader, int64(this.maxBodySize))
	}
	contentEncoding := strings.ToLower(resp.Header.Get("Content-Encoding"))
	if !resp.Uncompressed && (strings.Contains(contentEncoding, "gzip") || (contentEncoding == "" && strings.Contains(strings.ToLower((resp.Header.Get("Content-Type"))), "gzip"))) {
		bodyReader, err := gzip.NewReader(bodyReader)
		if err != nil {
			this.err = err
			return err
		}
		resp.Body = ioutil.NopCloser(bodyReader)
	}

	this.response = resp

	return nil
}

func (this *Request) download(file *os.File) error {
	readBytes := make([]byte, 1024)
	bodyReader := this.response.Body
	defer bodyReader.Close()

	total := this.response.ContentLength
	var current int64
	var lastTime time.Time

	defer func() {
		this.handleOnDownload(current, total)
	}()

	for {
		n, err := bodyReader.Read(readBytes)
		if n > 0 {
			if file != nil {
				if _, err := file.Write(readBytes[:n]); err != nil {
					return err
				}
			}
			current += int64(n)
			nowTime := time.Now()
			if nowTime.Sub(lastTime) > this.downloadProgressInterval {
				lastTime = nowTime
				this.handleOnDownload(current, total)
			}
			if current >= total {
				break
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	return nil
}

// OnRequest registers a function.
// Function will be executed on every
// request made by the Collector
func (this *Request) OnRequest(f RequestCallback) {
	this.lock.Lock()
	if this.requestCallbacks == nil {
		this.requestCallbacks = make([]RequestCallback, 0, 4)
	}
	this.requestCallbacks = append(this.requestCallbacks, f)
	this.lock.Unlock()
}
func (this *Request) handleOnRequest(req *http.Request) {
	if this.debug {

	}
	for _, f := range this.requestCallbacks {
		f(req)
	}
}

func (this *Request) OnDownload(f DownloadCallback) {
	this.lock.Lock()
	if this.downloadCallbacks == nil {
		this.downloadCallbacks = make([]DownloadCallback, 0, 4)
	}
	this.downloadCallbacks = append(this.downloadCallbacks, f)
	this.lock.Unlock()
}
func (this *Request) handleOnDownload(current, total int64) {
	if this.debug {

	}
	for _, f := range this.downloadCallbacks {
		f(current, total)
	}
}

func (this *Request) getClient() *http.Client {
	if this.client != nil {
		return this.client
	}

	if this.defaultClient == nil {
		this.defaultClient = newClient()
	}

	var jar http.CookieJar
	if this.useCookie {
		if this.defaultCookieJar == nil {
			this.defaultCookieJar, _ = cookiejar.New(nil)
		}
		jar = this.defaultCookieJar
	}
	this.defaultClient.Jar = jar

	if this.clientTimeout >= time.Duration(0) && this.clientTimeout != this.defaultClient.Timeout {
		this.defaultClient.Timeout = this.clientTimeout
	}

	trans, _ := this.defaultClient.Transport.(*http.Transport)
	if trans != nil {
		if this.connectTimeout == time.Duration(0) {
			this.connectTimeout = 30 * time.Second
		}
		if this.readWriteTimeout == time.Duration(0) {
			this.readWriteTimeout = 30 * time.Second
		}
		trans.DialContext = (&net.Dialer{
			Timeout:   this.connectTimeout,
			KeepAlive: this.readWriteTimeout,
			DualStack: true,
		}).DialContext

		if this.proxy != nil {
			trans.Proxy = this.proxy
		}

		trans.TLSClientConfig.InsecureSkipVerify = this.insecureTLSSkipVerify
	}

	return this.defaultClient
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
		// 建立TLS连接的时候，不验证服务器的证书
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
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

func createFormReader(data map[string]string) io.Reader {
	form := url.Values{}
	for k, v := range data {
		form.Add(k, v)
	}
	return strings.NewReader(form.Encode())
}

func createMultipartReader(boundary string, data map[string][]byte) io.Reader {
	dashBoundary := "--" + boundary

	body := []byte{}
	buffer := bytes.NewBuffer(body)

	buffer.WriteString("Content-type: multipart/form-data; boundary=" + boundary + "\n\n")
	for contentType, content := range data {
		buffer.WriteString(dashBoundary + "\n")
		buffer.WriteString("Content-Disposition: form-data; name=" + contentType + "\n")
		buffer.WriteString(fmt.Sprintf("Content-Length: %d \n\n", len(content)))
		buffer.Write(content)
		buffer.WriteString("\n")
	}
	buffer.WriteString(dashBoundary + "--\n\n")
	return buffer
}

// randomBoundary was borrowed from
// github.com/golang/go/mime/multipart/writer.go#randomBoundary
func randomBoundary() string {
	var buf [30]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf[:])
}

// 自动转成 utf-8
func (this *Request) fixCharset(html string) (string, error) {
	contentType := ""
	charset := htmlCharset(html)
	if charset == "utf-8" || charset == "utf8" {
		title := htmlTitle(html)
		if isGarbled(title) {
			charset = ""
		}
	}

	if charset == "" {
		if this.response == nil {
			return html, nil
		}

		contentType = strings.ToLower(this.response.Header.Get("Content-Type"))
	} else {
		contentType = "text/html; charset=" + charset
	}

	if contentType == "" ||
		strings.Contains(contentType, "utf-8") ||
		strings.Contains(contentType, "utf8") ||
		strings.Contains(contentType, "image/") ||
		strings.Contains(contentType, "video/") ||
		strings.Contains(contentType, "audio/") ||
		strings.Contains(contentType, "font/") {
		return html, nil
	}

	ret, err := encodeBytes([]byte(html), contentType)
	if err != nil {
		return html, err
	}

	return string(ret), nil
}

func encodeBytes(b []byte, contentType string) ([]byte, error) {
	r, err := charset.NewReader(bytes.NewReader(b), contentType)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}

// 获取 HTML 编码
func htmlCharset(html string) string {
	re, err := regexp.Compile(`(?i)<meta.+?charset=["|']*([\w|\-]+)["|'|;]*\s*.*?>`)
	if err != nil {
		return ""
	}

	charset := ""

	ar := re.FindStringSubmatch(html)
	if len(ar) > 0 {
		charset = strings.ToLower(ar[1])
	}

	return charset
}

// 获取 HTML Title
func htmlTitle(html string) string {
	re, err := regexp.Compile(`(?i)<title>(.*?)</title>`)
	if err != nil {
		return ""
	}

	title := ""
	ar := re.FindStringSubmatch(html)
	if len(ar) > 0 {
		title = strings.ToLower(ar[1])
	}

	return title
}

// 是否乱码
func isGarbled(str string) bool {
	if str == "" {
		return false
	}

	var i, n int
	ss := []rune(str)
	var alphaDashPattern = regexp.MustCompile(`[\w]`)
	for _, s := range ss {
		isAlpha := alphaDashPattern.MatchString(string(s))
		if isAlpha || unicode.Is(unicode.Scripts["Han"], s) {
			n++
		}
		i++
	}

	return n < int(math.Ceil(float64(i)*0.7))
}
