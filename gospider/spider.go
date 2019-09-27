package gospider

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nsqio/go-diskqueue"

	"github.com/zhuomouren/gohelpers"
	"github.com/zhuomouren/gohelpers/gonet"

	bolt "go.etcd.io/bbolt"
)

const (
	StatusPending = iota // 0
	StatusProcessing
	StatusSuspend
	StatusExiting
	StatusInvalid
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
	depth            int
	exiting          bool
	exitChan         chan bool
	db               *bolt.DB
	status           int
}

func New(name, url string) *GoSpider {
	this := &GoSpider{
		name:   name,
		url:    url,
		status: StatusPending,
	}

	this.visitCallbacks = make(map[string]VisitCallback)
	this.headerMap = map[string]string{}
	this.visitedUrls = make(map[string]bool)
	this.visitedCallbacks = make([]VisitedCallback, 0)
	this.waitCount = 0
	this.runCount = 0
	this.lock = &sync.RWMutex{}
	this.depth = 0
	this.logf = func(lvl diskqueue.LogLevel, f string, args ...interface{}) {
		log.Println(fmt.Sprintf(lvl.String()+" "+f, args...))
	}
	this.exitChan = make(chan bool)

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
	if this.queue != nil {
		return
	}

	if this.queueDataPath == "" {
		this.queueDataPath = "queuedata"
	}
	this.initQueue()

	if this.queue.Depth() == 0 {
		this.putQueue(this.getQueueData(1, this.url))
	}

	this.status = StatusProcessing

	go this.run()
}

func (this *GoSpider) Stop() error {
	// defer func() {
	// 	this.db.Close()
	this.status = StatusSuspend
	// 	this.exitChan <- true
	// }()

	if err := this.queue.Close(); err != nil {
		return err
	}

	return nil
}

func (this *GoSpider) Close() error {
	defer func() {
		// this.db.Close()
		this.status = StatusExiting
		this.exitChan <- true
	}()

	if err := this.queue.Empty(); err != nil {
		return err
	}
	if err := this.db.Close(); err != nil {
		return err
	}
	if err := os.Remove(this.dbfile()); err != nil {
		return err
	}

	return nil
}

func (this *GoSpider) RunCount() int64 {
	return this.runCount
}

func (this *GoSpider) Size() int64 {
	return this.queue.Depth()
}

func (this *GoSpider) Status() int {
	return this.status
}

// 初始化
func (this *GoSpider) initQueue() {
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

	var err error
	this.db, err = bolt.Open(this.dbfile(), 0600, nil)
	if err != nil {
		this.logf(diskqueue.ERROR, "bbolt err:%s", err.Error())
		return
	}
}

func (this *GoSpider) dbfile() string {
	return filepath.Join(this.queueDataPath, "history.db")
}

func (this *GoSpider) addHistory(url string) error {
	if this.db == nil {
		return nil
	}

	return this.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(getBucket(url))
		if err != nil {
			fmt.Println("err:", err.Error())
		}

		return b.Put([]byte(url), []byte(strconv.FormatBool(true)))
	})
}

func (this *GoSpider) visited(url string) bool {
	if this.db == nil {
		return false
	}

	var val bool
	this.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(getBucket(url))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(url))
		val = v != nil
		return nil
	})

	return val
}

func (this *GoSpider) run() {
	for {
		if this.queue.Depth() == 0 {
			if this.waitCount <= 3 {
				time.Sleep(5 * time.Second)
				this.waitCount++
				fmt.Println("this.waitCount: ", this.waitCount)
			} else {
				goto exit
			}
		} else {
			if err := this.runOne(); err != nil {
				this.logf(diskqueue.ERROR, "run err: %s", err.Error())
			}
			this.runCount++
		}
	}

exit:
	this.status = StatusExiting
	this.exitChan <- true
	return
}

func (this *GoSpider) putQueue(data []byte) {
	if err := this.queue.Put(data); err != nil {
		this.logf(diskqueue.ERROR, "queue put [%s] err: %s", string(data), err.Error())
		return
	}
}

func (this *GoSpider) runOne() error {
	data, ok := <-this.queue.ReadChan()
	if !ok {
		return fmt.Errorf("no data.")
	}

	depth, val := this.parseQueueData(data)
	if depth == 0 || val == "" {
		return fmt.Errorf("Cannot parse queue data: %s", string(data))
	}

	if this.depth > 0 && depth > this.depth {
		return nil
	}

	url := string(val)
	if this.visited(url) {
		return nil
	}
	this.addHistory(url)
	// if err := this.addHistory(url); err != nil {
	// 	return err
	// }
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

func getBucket(url string) []byte {
	h := md5.New()
	io.WriteString(h, url)
	val := hex.EncodeToString(h.Sum(nil))
	return []byte(val[:2])
}
