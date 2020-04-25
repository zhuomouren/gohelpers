package gospider

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/zhuomouren/gohelpers/golog"

	"github.com/zhuomouren/gohelpers"
	"github.com/zhuomouren/gohelpers/gonet"
	"github.com/zhuomouren/gohelpers/goqueue"

	bolt "go.etcd.io/bbolt"
)

const (
	StatusPending = iota // 0
	StatusProcessing
	StatusSuspend
	StatusExiting
	StatusInvalid
)

const (
	sep = "@@"
)

var (
	logger = golog.New("gospider")
)

type VisitCallback func(url, html string)

// 已经采集过的 URL，将不会放入队列
type VisitedCallback func(url string) bool

type GoSpider struct {
	name             string
	url              string
	charset          string
	proxy            string
	queue            *goqueue.Queue
	queueDataPath    string
	urlsRule         []string
	visitCallbacks   map[string]VisitCallback
	headerMap        map[string]string
	visitedCallbacks []VisitedCallback
	visitedUrls      map[string]bool
	waitCount        int
	runCount         int64
	http             *gonet.Request
	lock             *sync.RWMutex
	depth            int
	exiting          bool
	exitChan         chan bool
	db               *bolt.DB
	status           int
	sleep            time.Duration
	sep              string
}

func New(name, url string) *GoSpider {
	logger.Debug("new gospider",
		logger.String("name", name),
		logger.String("url", url),
	)
	this := &GoSpider{
		name:   name,
		url:    url,
		status: StatusPending,
	}

	this.visitCallbacks = make(map[string]VisitCallback)
	this.headerMap = map[string]string{}
	this.visitedUrls = make(map[string]bool)
	this.visitedCallbacks = make([]VisitedCallback, 0)
	this.waitCount = 1
	this.runCount = 0
	this.lock = &sync.RWMutex{}
	this.depth = 0
	this.exitChan = make(chan bool)
	this.sleep = 1 * time.Second
	this.sep = sep

	this.http = gonet.NewRequest()

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
	<-this.exitChan
}

func (this *GoSpider) Proxy(proxy string) *GoSpider {
	this.proxy = proxy
	return this
}

func (this *GoSpider) Depth(depth int) *GoSpider {
	this.depth = depth
	return this
}

func (this *GoSpider) Sleep(sleep time.Duration) *GoSpider {
	this.sleep = sleep
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
	this.visitCallbacks[gohelpers.String.DeepProcessingRegex(rule)] = f
	this.lock.Unlock()
}

func (this *GoSpider) handleOnVisit(url, html string) {
	for rule, f := range this.visitCallbacks {
		if this.exactMatch(rule, url) {
			logger.Debug("match url",
				logger.String("url", url),
				logger.String("match", "yes"),
				logger.String("rule", rule),
			)
			f(url, html)
		} else {
			logger.Debug("match url",
				logger.String("url", url),
				logger.String("match", "yes"),
			)
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
	logger.Info("gospider run",
		logger.String("name", this.name),
	)
	if this.status == StatusExiting {
		logger.Info("gospider has exited",
			logger.String("name", this.name),
		)
		return
	}

	// 防止重复运行
	if this.queue != nil {
		if this.status == StatusSuspend {
			this.status = StatusPending
			go this.run()
		}

		return
	}

	if this.queueDataPath == "" {
		this.queueDataPath = "queuedata"
	}

	logger.Debug("gospider init queue",
		logger.String("name", this.name),
	)
	this.initQueue()
	logger.Debug("gospider init queue completed",
		logger.String("name", this.name),
	)

	logger.Debug("gospider queue info: %d",
		logger.String("name", this.name),
		logger.Int("size", this.queue.Size()),
	)
	if this.queue.Size() == 0 {
		logger.Debug("add first url",
			logger.String("name", this.name),
			logger.String("url", this.url),
		)
		this.putQueue(this.getQueueData(1, this.url))
	}

	logger.Debug("status is processing",
		logger.String("name", this.name),
	)
	this.status = StatusProcessing

	go this.run()
}

func (this *GoSpider) Stop() {
	logger.Debug("gospider stop",
		logger.String("name", this.name),
	)
	this.status = StatusSuspend
}

func (this *GoSpider) Close() error {
	logger.Debug("gospider close",
		logger.String("name", this.name),
	)
	defer func() {
		this.status = StatusExiting
		this.exitChan <- true
		this.queue = nil
	}()

	if err := this.queue.Close(); err != nil {
		logger.Error("gospider close error",
			logger.String("name", this.name),
			logger.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func (this *GoSpider) RunCount() int64 {
	return this.runCount
}

func (this *GoSpider) Size() int {
	return this.queue.Size()
}

func (this *GoSpider) Status() int {
	return this.status
}

// 初始化
func (this *GoSpider) initQueue() {
	if this.name == "" || this.url == "" {
		return
	}

	queue, err := goqueue.New(this.name, this.queueDataPath)
	if err != nil {
		logger.Error("init queue error",
			logger.String("name", this.name),
			logger.String("error", err.Error()),
		)
		return
	}

	if len(this.sep) > 0 {
		queue.SetSeparator(this.sep)
	}

	this.queue = queue
}

func (this *GoSpider) run() {
	for {
		if this.status == StatusSuspend {
			this.exitChan <- true
			return
		}
		if this.status == StatusExiting {
			goto exit
		}
		if this.queue.Size() == 0 {
			if this.waitCount <= 3 {
				logger.Info("waiting for times",
					logger.Int("times", this.waitCount),
				)
				time.Sleep(5 * time.Second)
				this.waitCount++
			} else {
				goto exit
			}
		} else {
			if err := this.runOne(); err != nil {
				logger.Error("gospider run error",
					logger.String("name", this.name),
					logger.String("error", err.Error()),
				)
			}
			this.runCount++
		}
	}

exit:
	logger.Info("exiting ...",
		logger.String("name", this.name),
	)
	this.status = StatusExiting
	this.exitChan <- true
	return
}

func (this *GoSpider) putQueue(data string) {
	logger.Debug("gospider put queue",
		logger.String("name", this.name),
		logger.String("data", data),
	)
	if err := this.queue.Put(data); err != nil {
		logger.Error("gospider put queue error",
			logger.String("name", this.name),
			logger.String("data", data),
			logger.String("error", err.Error()),
		)
		return
	}
}

func (this *GoSpider) runOne() error {
	data, err := this.queue.Get()
	if err != nil {
		return err
	}

	depth, url := this.parseQueueData(data)
	if depth == 0 || url == "" {
		return fmt.Errorf("Cannot parse queue data: %s", data)
	}

	if this.depth > 0 && depth > this.depth {
		return nil
	}

	time.Sleep(this.sleep)
	html, err := this.getHTML(url)
	if err != nil {
		return err
	}

	this.handleOnVisit(url, html)

	nextDepth := depth + 1
	if this.depth > 0 && nextDepth > this.depth+1 {
		return nil
	}

	var urls []string
	links, err := gohelpers.String.GetLinks(html)
	if err != nil {
		return nil
	}

	for _, link := range links {
		absLink, err := gohelpers.URL.AbsoluteURL(link, url)
		if err != nil {
			// return err
			continue
		}

		if this.handleOnVisited(absLink) {
			continue
		}

		urls = append(urls, absLink)
	}

	gohelpers.String.RemoveDuplicate(&urls)
	// fmt.Println("urls:", urls)
	for _, link := range urls {
		for _, urlRule := range this.urlsRule {
			if this.exactMatch(urlRule, link) {
				this.putQueue(this.getQueueData(nextDepth, link))
			}
		}
	}

	return nil
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

func (this *GoSpider) getQueueData(depth int, url string) string {
	return fmt.Sprintf("%d%s%s", depth, this.sep, url)
}

func (this *GoSpider) parseQueueData(data string) (int, string) {
	parts := strings.SplitN(data, this.sep, 2)
	if len(parts) != 2 {
		return 0, ""
	}

	return gohelpers.Value(parts[0]).Int(), parts[1]
}
