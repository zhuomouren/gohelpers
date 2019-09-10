package gospider

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/nsqio/go-diskqueue"

	"github.com/zhuomouren/gohelpers"
	"github.com/zhuomouren/gohelpers/gonet"
)

type VisitCallback func(url, html string)

// 已经采集过的 URL，将不会放入队列
type VisitedCallback func(url string) bool

type GoSpider struct {
	name             string
	url              string
	charset          string
	proxy            string
	queue            diskqueue.Interface
	queueDataPath    string
	urlsRule         []string
	visitCallbacks   map[string]VisitCallback
	headerMap        map[string]string
	visitedCallbacks []VisitedCallback
	visitedUrls      map[string]bool
	waitCount        int
	runCount         int64
	logf             diskqueue.AppLogFunc
	http             *gonet.Request
	lock             *sync.RWMutex
	wg               *sync.WaitGroup
	depth            int
	exiting          bool
}

func New(name, url string) *GoSpider {
	this := &GoSpider{
		name: name,
		url:  url,
	}
	this.init()

	if this.queue.Depth() == 0 {
		this.putQueue(this.getQueueData(1, this.url))
	}

	return this
}

func (this *GoSpider) Charset(charset string) *GoSpider {
	this.charset = charset
	return this
}

func (this *GoSpider) DataPath(queueDataPath string) *GoSpider {
	this.queueDataPath = queueDataPath
	return this
}

func (this *GoSpider) Wait() {
	this.wg.Wait()
}

func (this *GoSpider) Proxy(proxy string) *GoSpider {
	this.proxy = proxy
	return this
}

func (this *GoSpider) Depth(depth int) *GoSpider {
	this.depth = depth
	return this
}

func (this *GoSpider) AddHeader(name, val string) *GoSpider {
	this.headerMap[name] = val
	return this
}

func (this *GoSpider) AddURLRule(rule string) *GoSpider {
	this.urlsRule = append(this.urlsRule, gohelpers.String.DeepProcessingRegex(rule))
	return this
}

func (this *GoSpider) URLRules(rules []string) *GoSpider {
	for _, rule := range rules {
		this.urlsRule = append(this.urlsRule, gohelpers.String.DeepProcessingRegex(rule))
	}

	return this
}

func (this *GoSpider) OnVisit(rule string, f VisitCallback) {
	this.lock.Lock()
	if this.visitCallbacks == nil {
		this.visitCallbacks = make(map[string]VisitCallback)
	}
	this.visitCallbacks[gohelpers.String.DeepProcessingRegex(rule)] = f
	this.lock.Unlock()
}

func (this *GoSpider) handleOnVisit(url, html string) {
	for rule, f := range this.visitCallbacks {
		if this.exactMatch(rule, url) {
			f(url, html)
		}
	}
}

func (this *GoSpider) OnVisited(f VisitedCallback) {
	this.lock.Lock()
	this.visitedCallbacks = append(this.visitedCallbacks, f)
	this.lock.Unlock()
}

func (this *GoSpider) handleOnVisited(url string) bool {
	for _, f := range this.visitedCallbacks {
		if b := f(url); b {
			return true
		}
	}

	return false
}

func (this *GoSpider) Run() {
	go this.run()
}

func (this *GoSpider) Stop() error {
	this.wgReset()
	return this.queue.Close()
}

func (this *GoSpider) Close() error {
	this.wgReset()
	return this.queue.Empty()
}

func (this *GoSpider) RunCount() int64 {
	return this.runCount
}

func (this *GoSpider) Size() int64 {
	return this.queue.Depth()
}

// 初始化
func (this *GoSpider) init() {
	if this.name == "" || this.url == "" {
		return
	}

	this.queue = diskqueue.New(
		this.name,
		this.queueDataPath,
		1024*1024*10,
		10,
		255,
		1024,
		2*time.Second,
		this.logf,
	)
	this.visitCallbacks = make(map[string]VisitCallback)
	this.headerMap = map[string]string{}
	this.visitedUrls = make(map[string]bool)
	this.visitedCallbacks = []VisitedCallback{}
	this.waitCount = 0
	this.runCount = 0
	this.wg = &sync.WaitGroup{}
	this.depth = 0

	if this.queue.Depth() > 0 {
		this.wg.Add(int(this.queue.Depth()))
	}

	this.http = gonet.NewRequest()
}

func (this *GoSpider) run() {
	for {
		if this.queue.Depth() == 0 {
			if this.waitCount <= 3 {
				this.wg.Add(1)
				time.Sleep(5 * time.Second)
				this.waitCount++
			} else {
				goto exit
			}
		}

		this.runOne()
		this.runCount++
	}

exit:
	// this.wg = &sync.WaitGroup{}
	return
}

func (this *GoSpider) putQueue(data []byte) {
	if err := this.queue.Put(data); err != nil {
		this.logf(diskqueue.ERROR, "queue put [%s] err: %s", string(data), err.Error())
		return
	}

	this.wg.Add(1)
}

func (this *GoSpider) runOne() error {
	defer this.wg.Done()
	data, ok := <-this.queue.ReadChan()
	if !ok {
		return nil
	}

	depth, val := this.parseQueueData(data)
	if depth == 0 || val == "" {
		return fmt.Errorf("Cannot parse queue data: %s", string(data))
	}

	if depth > this.depth {
		return nil
	}

	url := string(val)

	html, err := this.getHTML(url)
	if err != nil {
		return err
	}

	this.handleOnVisit(url, html)

	nextDepth := depth + 1
	if nextDepth > this.depth+1 {
		return nil
	}

	urls := make([]string, 0)
	links, err := gohelpers.String.GetLinks(html)
	if err != nil {
		return nil
	}
	for _, link := range links {
		absLink, err := gohelpers.URL.AbsoluteURL(link, url)
		if err != nil {
			return err
		}
		if this.handleOnVisited(absLink) {
			continue
		}

		urls = append(urls, link)
	}

	for _, link := range urls {
		for _, urlRule := range this.urlsRule {
			if this.exactMatch(urlRule, link) {
				this.putQueue(this.getQueueData(nextDepth, link))
			}
		}
	}

	return nil
}

func (this *GoSpider) wgReset() {
	this.wg = &sync.WaitGroup{}
}

func (this *GoSpider) getHTML(url string) (string, error) {
	if this.charset != "" {
		this.http.SetCharacterEncoding(this.charset)
	}
	if this.proxy != "" {
		this.http.SetProxyURL(this.proxy)
	}
	if len(this.headerMap) > 0 {
		for key, value := range this.headerMap {
			this.http.AddHeader(key, value)
		}
	}

	return this.http.GET(url).String()
}

// 精确匹配
func (this *GoSpider) exactMatch(regex, data string) bool {
	re, err := regexp.Compile(`(?i)` + regex)
	if err != nil {
		return false
	}

	str := re.FindString(data)
	return strings.EqualFold(str, data)
}

func (this *GoSpider) getQueueData(depth int, url string) []byte {
	return []byte(fmt.Sprintf("%d@@%s", depth, url))
}

func (this *GoSpider) parseQueueData(data []byte) (int, string) {
	val := string(data)
	parts := strings.SplitN(val, "@@", 2)
	if len(parts) != 2 {
		return 0, ""
	}

	return gohelpers.Value(parts[0]).Int(), parts[1]
}
