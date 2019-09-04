package gospider

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/imroc/req"
	"github.com/zhuomouren/gohelpers"
	"github.com/zhuomouren/gohelpers/goqueue"
)

type ParseFunc func(url, html string)

// 已经采集过的 URL，将不会放入队列
type VisitedCallback func(url string) bool

type GoSpider struct {
	Name            string
	URL             string
	Charset         string
	Proxy           string
	Queue           *goqueue.GoQueue
	urlsRule        []string
	parseRule       map[string]ParseFunc
	headerMap       map[string]string
	visitedCallback VisitedCallback
	visitedUrls     map[string]bool
	waitCount       int
	runCount        int64
	logf            goqueue.AppLogFunc
	request         *req.Req
}

func NewGoSpider(name, url string, options ...func(*GoSpider)) *GoSpider {
	ls := &GoSpider{
		Name: name,
		URL:  url,
	}
	ls.Init()

	for _, f := range options {
		f(ls)
	}

	if ls.Queue.Size() == 0 {
		ls.Queue.Put([]byte(ls.URL))
	}

	return ls
}

func Name(name string) func(*GoSpider) {
	return func(ls *GoSpider) {
		ls.Name = name
	}
}

func URL(url string) func(*GoSpider) {
	return func(ls *GoSpider) {
		ls.URL = url
	}
}

func Charset(charset string) func(*GoSpider) {
	return func(ls *GoSpider) {
		ls.Charset = charset
	}
}

func Proxy(proxy string) func(*GoSpider) {
	return func(ls *GoSpider) {
		ls.Proxy = proxy
	}
}

func Queue(queue *goqueue.GoQueue) func(*GoSpider) {
	return func(ls *GoSpider) {
		ls.Queue = queue
	}
}

// 初始化
func (ls *GoSpider) Init() {
	if ls.Name == "" || ls.URL == "" {
		return
	}

	ls.Queue = goqueue.NewGoQueue(ls.URL)
	ls.parseRule = make(map[string]ParseFunc)
	ls.headerMap = map[string]string{}
	ls.visitedUrls = make(map[string]bool)
	ls.visitedCallback = func(url string) bool {
		return false
	}
	ls.waitCount = 0
	ls.runCount = 0

	ls.request = req.New()
}

func (ls *GoSpider) AddHeader(name, val string) {
	ls.headerMap[name] = val
}

func (ls *GoSpider) Visited(visited VisitedCallback) {
	ls.visitedCallback = visited
}

func (ls *GoSpider) AddURL(rule string) {
	ls.urlsRule = append(ls.urlsRule, gohelpers.String.DeepProcessingRegex(rule))
}

func (ls *GoSpider) Parse(rule string, parseFunc ParseFunc) {
	ls.parseRule[gohelpers.String.DeepProcessingRegex(rule)] = parseFunc
}

func (ls *GoSpider) Run() {
	for {
		if ls.Queue.Size() == 0 {
			if ls.waitCount <= 3 {
				time.Sleep(5 * time.Second)
				ls.waitCount++
			} else {
				goto exit
			}
		}

		ls.run()
		ls.runCount++
	}

exit:
	return
}

func (ls *GoSpider) RunCount() int64 {
	return ls.runCount
}

func (ls *GoSpider) run() error {
	val, ok := ls.Queue.Get()
	if !ok {
		return nil
	}

	url := string(val)

	html, err := ls.getHTML(url)
	if err != nil {
		return err
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
		if ls.visitedCallback(absLink) {
			continue
		}

		if _, ok := ls.visitedUrls[absLink]; ok {
			continue
		}

		urls = append(urls, link)
	}

	for _, link := range urls {
		for _, urlRule := range ls.urlsRule {
			if ls.exactMatch(urlRule, link) {
				if err := ls.Queue.Put([]byte(link)); err == nil {
					ls.visitedUrls[link] = true
				} else {
					fmt.Sprintf("queue put [%s] err: %s", link, err.Error())
				}
			}
		}
	}

	for rule, parseFunc := range ls.parseRule {
		if ls.exactMatch(rule, url) {
			parseFunc(url, html)
		}
	}

	return nil
}

func (ls *GoSpider) getHTML(url string) (string, error) {
	// html, err := GetHTMLFromURL(url, ls.Charset, ls.Proxy, ls.headerMap, 30)
	// if err != nil {
	// 	fmt.Println("GetHTMLFromURL err: ", err.Error())
	// 	return
	// }

	return "", nil
}

// 精确匹配
func (ls *GoSpider) exactMatch(regex, data string) bool {
	re, err := regexp.Compile(`(?i)` + regex)
	if err != nil {
		return false
	}

	str := re.FindString(data)
	return strings.EqualFold(str, data)
}
