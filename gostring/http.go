package gostring

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/mozillazg/go-pinyin"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// email 反爬虫
func (this *GoString) AntiSpamBot(email string) string {
	char := []rune(email)
	rand.Seed(time.Now().Unix())
	var out string
	var i int
	for _, c := range char {
		i = rand.Intn(2)
		if i == 1 {
			out += fmt.Sprintf("&#%d;", int(c))
		} else {
			out += string(c)
		}
	}

	return out
}

// 获取汉字的拼音
func (this *GoString) Pinyin(text, sep string) string {
	result := ""
	for _, r := range text {
		// 如果是中文
		if unicode.Is(unicode.Scripts["Han"], r) {
			result = sep + result + getPinyin(string(r)) + sep
		} else {
			result += string(r)
		}
	}

	if sep != "" {
		result = strings.TrimPrefix(result, sep)
		result = strings.TrimSuffix(result, sep)
	}

	return result
}

// 获取汉字的拼音
func getPinyin(text string) string {
	arg := pinyin.NewArgs()
	str := pinyin.Slug(text, arg)
	return strings.Replace(str, "-", "", -1)
}

// 获取汉字的首字母
func (this *GoString) FirstLetter(text, sep string) string {
	arg := pinyin.NewArgs()
	arg.Style = pinyin.FirstLetter
	str := pinyin.Pinyin(text, arg)
	if len(str) > 0 {
		result := ""
		for _, s := range str {
			result = result + s[0] + sep
		}
		return strings.TrimSuffix(result, sep)
	}

	return ""
}

func (this *GoString) StripHTML(html string) string {
	if html == "" {
		return html
	}

	content := strings.TrimSpace(html) //去空格

	content = strings.NewReplacer(
		`&amp;`, "&",
		`&#38;`, "&",
		`&#x26;`, "&",
		`&#39;`, "'",
		`&lt;`, "<",
		`&#60;`, "<",
		`&#x3C;`, "<",
		`&gt;`, ">",
		`&#62;`, ">",
		`&#x3E;`, ">",
		`&#34;`, `"`,
		`&quot;`, `"`,
		`&#34;`, `"`,
	).Replace(content)

	//将HTML标签全转换成小写
	re, _ := regexp.Compile("\\<[\\S\\s]+?\\>")
	content = re.ReplaceAllStringFunc(content, strings.ToLower)
	//去除STYLE
	re, _ = regexp.Compile("\\<style[\\S\\s]+?\\</style\\>")
	content = re.ReplaceAllString(content, "")

	//去除SCRIPT
	re, _ = regexp.Compile("\\<script[\\S\\s]+?\\</script\\>")
	content = re.ReplaceAllString(content, "")

	re, _ = regexp.Compile("\\<h[\\S\\s]+?\\</h\\>")
	content = re.ReplaceAllString(content, "")
	for i := 1; i < 7; i++ {
		re, _ = regexp.Compile("\\<h" + strconv.Itoa(i) + "[\\S\\s]+?\\</h" + strconv.Itoa(i) + "\\>")
		content = re.ReplaceAllString(content, "")
	}

	r := strings.NewReplacer("<br>", "\r\n", "<br/>", "\r\n", "<br />", "\r\n", "<p>", "\r\n", "</p>", "\r\n")
	content = r.Replace(content)
	//去除连续的换行符
	re, _ = regexp.Compile("\\s{2,}")
	content = re.ReplaceAllString(content, "\r\n\r\n")

	//去除所有尖括号内的HTML代码，并换成换行符
	re, _ = regexp.Compile("\\<[\\S\\s]+?\\>")
	content = re.ReplaceAllString(content, "")

	content = strings.Trim(content, "\r\n")

	return content
}

// 自动转成 utf-8
func (this *GoString) AutoUTF8(html string) (string, error) {
	charset := this.HtmlCharset(html)
	if charset == "" {
		return html, errors.New("")
	}
	if strings.EqualFold(charset, "utf-8") || strings.EqualFold(charset, "utf8") {
		return html, nil
	}

	if strings.EqualFold(charset, "gbk") || strings.EqualFold(charset, "gb2312") {
		if ret, err := this.GBKToUTF8([]byte(html)); err != nil {
			return html, err
		} else {
			return string(ret), nil
		}
	}

	contentType := "text/html; charset=" + charset
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

// gbk 转 utf-8
func (this *GoString) GBKToUTF8(text []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(text), simplifiedchinese.GBK.NewDecoder())
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// utf-8 转 gbk
func (this *GoString) UTF8ToGBK(text []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(text), simplifiedchinese.GBK.NewEncoder())
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// 获取 HTML 编码
func (this *GoString) HtmlCharset(html string) string {
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

// 获取HTML中base标签指定的链接地址
func (this *GoString) HtmlBaseUrl(html string) string {
	return ""
}

func (this *GoString) HtmlAttributes(data, ele, attr string) ([]string, error) {
	attrs := make([]string, 0)

	root, err := html.Parse(strings.NewReader(data))
	if err != nil {
		return attrs, err
	}

	var linksNode func(*html.Node)
	linksNode = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == ele {
			attrs = append(attrs, strings.TrimSpace(getAttribute(n, attr)))
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			linksNode(c)
		}
	}
	linksNode(root)

	return attrs, nil
}

// 提取链接
func (this *GoString) GetLinks(data string) ([]string, error) {
	urls := make([]string, 0)
	re, err := regexp.Compile(`<style[\S\s]+?</style>`)
	if err != nil {
		return urls, err
	}
	data = re.ReplaceAllString(data, "")

	re, err = regexp.Compile(`<script[\S\s]+?</script>`)
	if err != nil {
		return urls, err
	}
	data = re.ReplaceAllString(data, "")

	root, err := html.Parse(strings.NewReader(data))
	if err != nil {
		return urls, err
	}

	var linksNode func(*html.Node)
	linksNode = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := strings.TrimSpace(getAttribute(n, "href"))
			if href != "" && href != "/" && href != "#" && !strings.Contains(strings.ToLower(href), "javascript:") {
				urls = append(urls, href)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			linksNode(c)
		}
	}
	linksNode(root)

	return urls, nil
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

// 去除 html标签
func (this *GoString) StripHTMLTags(html string) string {
	re, err := regexp.Compile("\\<[\\S\\s]+?\\>")
	if err != nil {
		return html
	}

	return re.ReplaceAllString(html, "")
}
