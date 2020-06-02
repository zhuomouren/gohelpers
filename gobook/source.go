package gobook

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"

	"github.com/zhuomouren/gohelpers/gonet"
	"golang.org/x/net/html"
)

type Source struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// 获取重定向之后的 URL
func GetLocationUrl(rawurl string) (string, error) {
	trans := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		//Proxy:               b.setting.Proxy,
		Dial: func(netw, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout(netw, addr, 30*time.Second)
			if err != nil {
				return nil, err
			}
			err = conn.SetDeadline(time.Now().Add(30 * time.Second))
			return conn, err
		},
		DisableKeepAlives:   true,
		MaxIdleConnsPerHost: -1,
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("Redirected!")
		},
		Transport: trans,
	}

	req, err := http.NewRequest("GET", rawurl, nil)
	if err != nil {
		return "", err
	}

	redir := ""

	resp, err := client.Do(req)
	if err != nil {
		if e, ok := err.(*url.Error); ok && e.Err != nil {
			if !strings.Contains(e.Err.Error(), "dial tcp") {
				redir = e.URL
			}
		} else {
			return "", err
		}
	}

	if resp != nil {
		defer resp.Body.Close()
	}

	if redir != "" {
		return redir, nil
	}

	switch resp.StatusCode {
	case http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusSeeOther,
		http.StatusTemporaryRedirect:
		redir = resp.Header.Get("Location")
	}

	return redir, nil
}

func GetOriginalUrlFromBaiduLink(source *Source, wg *sync.WaitGroup) {
	defer wg.Done()

	redirURL, err := GetLocationUrl(source.URL)
	if err != nil {
		log.Println("GetOriginalUrlFromBaiduLink:" + err.Error())
		return
	}

	source.URL = redirURL
}

func GetSourcesByBaidu(name string) ([]*Source, error) {
	var sources []*Source
	url := "https://www.baidu.com/s?wd=" + url.QueryEscape(name+"最新章节列表")

	data, err := gonet.NewRequest().AddHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9").SetUserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.122 Safari/537.36").GET(url).String()
	if err != nil {
		return nil, err
	}

	root, err := html.Parse(strings.NewReader(data))
	if err != nil {
		return nil, err
	}

	var linkNodes func(*html.Node)
	linkNodes = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.ToLower(n.Data) == "div" {
			attr := getAttribute(n, "data-tools")
			if attr != "" && strings.Contains(attr, "title") && strings.Contains(attr, "url") && strings.Contains(attr, "link?url") {
				title := gjson.Get(attr, "title").String()
				url := gjson.Get(attr, "url").String()
				if title != "" && strings.Contains(url, "link?url=") {
					sources = append(sources, &Source{Title: title, URL: url})
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			linkNodes(c)
		}
	}
	linkNodes(root)

	iNum := len(sources)
	if iNum >= 50 {
		iNum = 50
	}

	wg := &sync.WaitGroup{}
	for i := 0; i < iNum; i++ {
		wg.Add(1)
		go GetOriginalUrlFromBaiduLink(sources[i], wg)
	}
	wg.Wait()

	var results []*Source
	for _, source := range sources {
		if source.URL != "" && !isIgnore(source.URL) && getPath(source.URL) != "" {
			results = append(results, source)
		}
	}

	return results, nil
}

func GetSourcesFromBing(name string) ([]*Source, error) {
	var sources []*Source
	url := "https://www.baidu.com/s?wd=" + url.QueryEscape(name+"最新章节列表")

	data, err := gonet.NewRequest().SetUserAgent("Mozilla/5.0 (Windows NT 6.2; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/27.0.1453.94 Safari/537.36").GET(url).String()
	if err != nil {
		return nil, err
	}
	root, err := html.Parse(strings.NewReader(data))
	if err != nil {
		return nil, err
	}

	var linkNodes func(*html.Node)
	linkNodes = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.ToLower(n.Data) == "a" {
			attr := getAttribute(n, "data-click")
			href := getAttribute(n, "href")
			title := getInnerText(n)
			if attr != "" && strings.Contains(href, "link?url=") && strings.Contains(title, name) {
				sources = append(sources, &Source{Title: title, URL: href})
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			linkNodes(c)
		}
	}
	linkNodes(root)

	iNum := len(sources)
	if iNum >= 50 {
		iNum = 50
	}

	wg := &sync.WaitGroup{}
	for i := 0; i < iNum; i++ {
		wg.Add(1)
		go GetOriginalUrlFromBaiduLink(sources[i], wg)
	}
	wg.Wait()

	var results []*Source
	for _, source := range sources {
		if source.URL != "" && !isIgnore(source.URL) && getPath(source.URL) != "" {
			results = append(results, source)
		}
	}

	return results, nil
}
