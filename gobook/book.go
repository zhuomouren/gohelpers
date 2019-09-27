package gobook

import (
	"regexp"
	"strings"

	"github.com/zhuomouren/gohelpers/gonet"

	"github.com/zhuomouren/gohelpers"
)

const (
	StatusSerial = iota // 0
	StatusFinished
)

type Book struct {
	URL          string `json:"url"`
	Name         string `json:"name"`
	Author       string `json:"author"`
	Cover        string `json:"cover"`
	Category     string `json:"category"`
	Summary      string `json:"summary"`
	ChapterCount int    `json:"chapter_count"`
	ReadURL      string `json:"read_url"`
	Status       int    `json:"status"`
}

func GetBook(url string) *Book {
	html, err := gonet.NewRequest().GET(url).String()
	if err != nil {
		return nil
	}

	return GetBookByOGP(url, html)
}

func GetBookByOGP(url, html string) *Book {
	// 提取 head
	substr := "</head>"
	if strings.Contains(html, substr) {
		data := strings.Split(html, substr)
		if len(data) == 2 {
			html = data[0]
		}
	}

	book := &Book{}

	name := GetOpenGraphProtocol("og:novel:book_name", html)
	if name == "" {
		name = GetOpenGraphProtocol("og:title", html)
	}
	author := GetOpenGraphProtocol("og:novel:author", html)
	readURL := GetOpenGraphProtocol("og:novel:read_url", html)
	if !gohelpers.URL.HasPrefix(readURL) {
		readURL, _ = gohelpers.URL.AbsoluteURL(readURL, url)
	}

	if name == "" || author == "" || readURL == "" {
		return nil
	} else {
		book.Name = name
		book.Author = author
		book.ReadURL = readURL
	}

	cover := GetOpenGraphProtocol("og:image", html)
	// 应该都是绝对路径，没有这种情况
	if !gohelpers.URL.HasPrefix(cover) {
		cover, _ = gohelpers.URL.AbsoluteURL(cover, url)
	}
	book.Cover = cover
	book.Category = GetOpenGraphProtocol("og:novel:category", html)
	book.Summary = GetOpenGraphProtocol("og:description", html)
	status := GetOpenGraphProtocol("og:novel:status", html) // 写作进度
	if strings.Contains(status, "完") {
		book.Status = StatusFinished
	} else {
		book.Status = StatusSerial
	}

	return book
}

// 获取 Open Graph Protocol
func GetOpenGraphProtocol(tag, html string) string {
	re, err := regexp.Compile(`(?i)<meta\s*\b(property|name)\b=["|']` + tag + `["|']\s*content=["|']([\s\S]*?)["|'|;].*?>`)
	if err != nil {
		return ""
	}

	og := ""
	ret := re.FindStringSubmatch(html)
	if len(ret) == 3 {
		og = strings.ToLower(ret[2])
	}

	return og
}
