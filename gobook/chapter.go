package gobook

import (
	"bytes"
	"math"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/zhuomouren/gohelpers/gonet"
	"github.com/zhuomouren/gohelpers/goreadability"
	"golang.org/x/net/html"
)

type Chapter struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

func GetChapter(html, url string) *Chapter {
	r := goreadability.NewReadability(html, url)

	return &Chapter{
		Title: r.GetTitle(),
		Body:  r.GetContent(),
		URL:   url,
	}
}

func GetChapterByURL(url string) (*Chapter, error) {
	html, err := gonet.NewRequest().GET(url).String()
	if err != nil {
		return nil, err
	}

	return GetChapter(html, url), nil
}

///////////////////////////////////////////////////
//
// 小说章节列表提取函数
//
///////////////////////////////////////////////////

func GetChapters(s, u string) []*Chapter {
	urls := getLinks(s, u)
	maxCount := len(urls)
	if maxCount <= 0 {
		return nil
	}

	maybeUrls := make([]*Chapter, 0, maxCount) //, 1800
	for _, url := range urls {
		if maybeChapterTitle(url) {
			maybeUrls = append(maybeUrls, url)
		}
	}

	// 最少 5 章节
	if len(maybeUrls) <= 5 {
		return nil
	}

	maybeUrlsCount := int(math.Ceil(float64(len(maybeUrls)) / float64(2)))

	url_regex := make(map[string]int)
	original_args := map[string]map[string]int{}
	regex_args := map[string]map[string]int{}
	for _, maybeUrl := range maybeUrls {
		_url_regex, _original_args, _regex_args := extractRegex(maybeUrl.URL, false)
		if _, ok := url_regex[_url_regex]; ok {
			url_regex[_url_regex]++
		} else {
			url_regex[_url_regex] = 1
		}

		for key, value := range _original_args {
			if v, ok := original_args[key]; ok {
				if _v, _ok := v[value]; _ok {
					original_args[key][value] = _v + 1
				}
			} else {
				original_args[key] = map[string]int{value: 1}
			}
		}

		for key, value := range _regex_args {
			if v, ok := regex_args[key]; ok {
				if _v, _ok := v[value]; _ok {
					regex_args[key][value] = _v + 1
				}
			} else {
				regex_args[key] = map[string]int{value: 1}
			}
		}
	}

	regex := ""
	max := 0
	for _regex, n := range url_regex {
		if n > max {
			regex, max = _regex, n
		}
	}

	max = 0
	filename := ""
	if v, ok := original_args["__tp_filename"]; ok {
		for _v, n := range v {
			if n > max {
				filename, max = _v, n
			}
		}
	}

	if max > maybeUrlsCount {
		regex = strings.Replace(regex, "__tp_alnum__", filename, -1)
		regex = strings.Replace(regex, "__tp_alpha__", filename, -1)
	} else {
		regex = strings.Replace(regex, "__tp_alnum__", "([0-9]{1,})", -1)
		regex = strings.Replace(regex, "__tp_alpha__", "([^/]+)", -1)
	}

	parsedUrl, _ := url.Parse(regex)
	if parsedUrl.RawQuery != "" {
		query := make([]string, 0, 5)
		_original_args := parsedUrl.Query()
		for k, v := range _original_args {
			max = 0
			__key := ""
			if _v, ok := original_args[k]; ok {
				for __k, __v := range _v {
					if __v > max {
						__key, max = __k, __v
					}
				}
			}

			if max > maybeUrlsCount {
				query = append(query, k+"="+__key)
			} else {
				query = append(query, k+"="+v[0])
			}

			parsedUrl.RawQuery = strings.Join(query, "&")
		}
	}
	regex = parsedUrl.String()
	regex, _ = url.QueryUnescape(regex)
	//fmt.Println("regex: " + regex)

	maybe_urls := make([]map[string]*Chapter, 0, maxCount) //map[string]map[string]string{} // make([]map[string]string, 0, 1800)
	repeat_urls := make(map[string][]*Chapter)
	order := true
	for _, url := range urls {
		re, err := regexp.Compile(regex)
		if err != nil {
			continue
		}

		ar := re.FindStringSubmatch(url.URL)
		if len(ar) < 2 {
			continue
		}
		match := ar[1]
		if !isNumeric(match) {
			order = false
		}

		_b := false
		_del := false
		var ii int
		for i, _url := range maybe_urls {
			if _, ok := _url[match]; ok {
				_b = true
				if maybeChapterTitle(url) {
					repeat_urls[match] = append(repeat_urls[match], url)
					_del = true
					ii = i
				}
			}
		}
		if !_b {
			m := make(map[string]*Chapter)
			m[match] = url
			maybe_urls = append(maybe_urls, m)
		} else {
			if _del && ii >= 0 && len(maybe_urls) > ii {
				maybe_urls = append(maybe_urls[:ii], maybe_urls[ii+1:]...)
				m := make(map[string]*Chapter)
				m[match] = url
				maybe_urls = append(maybe_urls, m)
			}
		}
	}

	for key, value := range repeat_urls {
		arr := make([]*Chapter, 0, 5)
		for _, v := range value {
			if maybeChapterTitle(v) {
				arr = append(arr, v)
			}
		}

		if len(arr) > 0 {
			for i, m := range maybe_urls {
				if _, ok := m[key]; ok {
					_m := make(map[string]*Chapter)
					_m[key] = arr[len(arr)-1]
					maybe_urls[i] = _m
					break
				}
			}
		}
	}
	//fmt.Println("maybe_urls = %v", maybe_urls)
	//如果文件名全是数字，则排序
	// 采用冒泡排序算法
	if order {
		num := len(maybe_urls)
		for i := 0; i < num; i++ {
		exitFor:
			for j := i + 1; j < num; j++ {
				var err error
				var ii int
				for k, _ := range maybe_urls[i] {
					//ii, _ = strconv.Atoi(k)
					ii, err = strconv.Atoi(k)
					if err != nil {
						break exitFor
					}
					break
				}
				var jj int
				for _k, _ := range maybe_urls[j] {
					//jj, _ = strconv.Atoi(_k)
					jj, err = strconv.Atoi(_k)
					if err != nil {
						break exitFor
					}
					break
				}
				if ii > jj {
					temp := maybe_urls[i]
					maybe_urls[i] = maybe_urls[j]
					maybe_urls[j] = temp
				}
			}
		}
	}

	num := len(maybe_urls)
	chapterUrls := make([]*Chapter, 0, num)
	//循环排序
	for i := 0; i < num; i++ {
		for _, v := range maybe_urls[i] {
			if v.URL != "" {
				chapterUrls = append(chapterUrls, v)
			}
		}
	}

	return chapterUrls
}

func GetChaptersFromURL(url string) (urls []*Chapter) {
	data, err := gonet.NewRequest().GET(url).String()
	if err != nil {
		return
	}

	urls = GetChapters(data, url)
	return
}

func getLinks(data, url string) []*Chapter {
	urls := make([]*Chapter, 0) //, 1800
	re, _ := regexp.Compile(`<style[\S\s]+?</style>`)
	data = re.ReplaceAllString(data, "")

	re, _ = regexp.Compile(`<script[\S\s]+?</script>`)
	data = re.ReplaceAllString(data, "")

	root, err := html.Parse(strings.NewReader(data))
	if err != nil {
		//		fmt.Println("html parse error")
		return urls
	}

	var linksNode func(*html.Node)
	linksNode = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := strings.TrimSpace(getAttribute(n, "href"))
			if href != "" && href != "/" && href != "#" && !strings.Contains(strings.ToLower(href), "javascript:") {
				u := &Chapter{URL: rel2abs(href, url), Title: getInnerText(n)}
				urls = append(urls, u)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			linksNode(c)
		}
	}
	linksNode(root)

	return urls
}

func maybeChapterTitle(url *Chapter) bool {
	array := [31]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "零", "一", "二", "三", "四", "五", "六", "七", "八", "九", "十", "０", "１", "２", "３", "４", "５", "６", "７", "８", "９"}
	for _, v := range array {
		if strings.Contains(url.Title, v) {
			return true
		}
	}

	return false
}

func extractRegex(u string, fuzzy bool) (string, map[string]string, map[string]string) {
	tp_filename := ""
	parsedUrl, _ := url.Parse(u)
	if parsedUrl.Path != "" && parsedUrl.Path != "/" {
		backslash := false
		if last := parsedUrl.Path[len(parsedUrl.Path)-1]; last == '/' {
			backslash = true
		}
		urlSplit := strings.Split(strings.TrimRight(parsedUrl.Path, "/"), "/")
		url_split := urlSplit[:]
		filename := url_split[len(url_split)-1]
		url_split = url_split[:len(url_split)-1]
		ext := getExt(filename)
		b := false
		name := ""
		if ext != "" {
			if ok, e := regexp.MatchString("^[a-zA-Z]+$", ext); ok && e == nil {
				name = basename(filename)
				b = true
			}
		} else {
			name = filename
		}

		chars := [4]string{".", "_", "-", ","}
		has_char := false
		for _, char := range chars {
			if strings.Contains(name, char) {
				has_char = true
				__url_split := strings.Split(name, char)
				_url_split := __url_split[:]
				filename = _url_split[len(_url_split)-1]
				_url_split = _url_split[:len(_url_split)-1]
				// 模糊匹配
				if fuzzy {
					_filename := _url_split[len(_url_split)-1]
					_url_split = _url_split[:len(_url_split)-1]
					if ok, e := regexp.MatchString("^[0-9]+$", _filename); ok && e == nil {
						_url_split = append(_url_split, "__tp_alnum__")
					} else {
						_url_split = append(_url_split, "__tp_alpha__")
					}
				}
				if ok, e := regexp.MatchString("^[0-9]+$", filename); ok && e == nil {
					_url_split = append(_url_split, "__tp_alnum__")
				} else {
					_url_split = append(_url_split, "__tp_alpha__")
				}
				name = strings.Join(_url_split, char)
				tp_filename = filename
				break
			}
		}

		if !has_char {
			tp_filename = name
			if ok, e := regexp.MatchString("^[0-9]+$", name); ok && e == nil {
				name = "__tp_alnum__"
			} else {
				name = "__tp_alpha__"
			}
		}

		if b {
			url_split = append(url_split, name+"."+ext)
		} else {
			url_split = append(url_split, name)
		}

		parsedUrl.Path = strings.Join(url_split, "/")
		if backslash {
			parsedUrl.Path = strings.TrimRight(parsedUrl.Path, "/") + "/"
		}
	}

	original_args := make(map[string]string)
	regex_args := make(map[string]string)
	if parsedUrl.RawQuery != "" {
		query := make([]string, 0, 5)
		_original_args := parsedUrl.Query()
		for k, v := range _original_args {
			original_args[k] = v[0]
			if ok, e := regexp.MatchString("^[0-9]+$", v[0]); ok && e == nil {
				regex_args[k] = "([0-9]{1,})"
				query = append(query, k+"=([0-9]{1,})")
			} else {
				regex_args[k] = "([^/]+)"
				query = append(query, k+"=([^/]+)")
			}
		}
		parsedUrl.RawQuery = strings.Join(query, "&")
	}

	regex := parsedUrl.String()
	original_args["__tp_filename"] = tp_filename
	return regex, original_args, regex_args
}

/////////////////////////////////////////////////////
//
//
//
//////////////////////////////////////////////////////

func getPath(fromUrl string) string {
	u, _ := url.Parse(fromUrl)
	re, _ := regexp.Compile("/[^/]*$")
	return re.ReplaceAllString(u.Path, "")
}

func isIgnore(u string) bool {
	u = strings.ToLower(u)
	urls := [...]string{
		"qidian.com", "17k.com", "chuangshi.com", "yuncheng.com", "qq.com", "163.com",
		"sina.com", "sohu.com", "ifeng.com", "hao123.com", "baidu.com", "sogou.com",
		"qdmm.com", "douban.com", "2345.com", "zhulang.com", "zongheng.com", "bookso.net",
		"soduso.com", "readnovel.com"}
	for _, s := range urls {
		if strings.Contains(u, s) {
			return true
		}
	}

	return false
}

///////////////////////////////////////////////////
//
// 公共函数
//
///////////////////////////////////////////////////
func getInnerHTML(n *html.Node) string {
	var buf bytes.Buffer

	if n != nil {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			html.Render(&buf, c)
		}
	}

	return buf.String()
}

func getInnerText(n *html.Node) string {
	if n != nil {
		if n.Type == html.TextNode {
			return n.Data
		} else if n.FirstChild != nil {
			var buf bytes.Buffer
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				buf.WriteString(getInnerText(c))
			}
			return buf.String()
		}
	}

	return ""
}

func getElementsByTagName(n *html.Node, tag string) []*html.Node {
	var nodes []*html.Node
	var getElements func(*html.Node, string)
	getElements = func(node *html.Node, _tag string) {
		if node != nil {
			if node.Type == html.ElementNode && strings.ToLower(node.Data) == _tag {
				nodes = append(nodes, node)
			}

			for c := node.FirstChild; c != nil; c = c.NextSibling {
				getElements(c, _tag)
			}
		}
	}
	getElements(n, tag)

	return nodes
}

func getAttribute(n *html.Node, attr string) string {
	var val string = ""
	if n != nil {
		for _, a := range n.Attr {
			if a.Key == attr {
				val = a.Val
				break
			}
		}
	}

	return val
}

func setAttribute(n *html.Node, attr, value string) {
	if n != nil {
		for i, a := range n.Attr {
			if a.Key == attr {
				n.Attr[i].Val = value
				break
			}
		}
	}
}

func getElementHTML(n *html.Node) string {
	var content string
	var getHTML func(*html.Node)
	getHTML = func(node *html.Node) {
		if node != nil {
			if node.Type == html.ElementNode {
				content = content + getInnerHTML(node)
			}

			for c := node.FirstChild; c != nil; c = c.NextSibling {
				getHTML(c)
			}
		}
	}
	getHTML(n)

	return content
}

// 相对路径转绝对路径
func rel2abs(rel, base string) string {
	if rel == "" {
		return ""
	}

	u, err := url.Parse(rel)
	if err != nil {
		// fmt.Println("rel2abs: [rel " + rel + "] " + err.Error())
		return ""
	}

	if u.IsAbs() {
		return rel
	}

	if rel[0] == '#' || rel[0] == '?' {
		return base + rel
	}

	u, _ = url.Parse(base)
	path := u.Path
	if getExt(u.Path) != "" {
		re, _ := regexp.Compile("/[^/]*$")
		path = re.ReplaceAllString(u.Path, "")
	}

	var full string
	if rel[0] == '/' {
		full = u.Host + rel
	} else {
		full = u.Host + path + "/" + rel
	}

	var dst []string
	src := strings.Split(full, "/")
	for _, elem := range src {
		switch elem {
		case ".":
			// drop
		case "..":
			if len(dst) > 0 {
				dst = dst[:len(dst)-1]
			}
		default:
			dst = append(dst, elem)
		}
	}
	if last := src[len(src)-1]; last == "." || last == ".." {
		dst = append(dst, "")
	}

	return u.Scheme + "://" + strings.Replace(strings.TrimLeft(strings.Join(dst, "/"), "/"), "//", "/", -1)
}

func isNumeric(s string) bool {
	ok, _ := regexp.MatchString(`^[0-9]+$`, s)
	return ok
}

func getFilename(u string) string {
	return filepath.Base(u)
}

// 后缀不包含 .
func getExt(u string) string {
	return strings.TrimLeft(filepath.Ext(u), ".")
}

//
func basename(path string) string {
	filename := getFilename(path)
	ext := getExt(filename)
	if ext != "" {
		return filename[:len(filename)-(len(ext)+1)]
	}

	return filename
}
