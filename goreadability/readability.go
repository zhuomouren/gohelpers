package goreadability

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"math"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	//	"github.com/astaxie/beego"
)

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

// 后缀不包含 .
func getExt(u string) string {
	return strings.TrimLeft(filepath.Ext(u), ".")
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

// 求最大公共子串，支持中文
func longestCommonSubstring(str1, str2 string) int {
	s1, s2 := []rune(str1), []rune(str2)
	size1, size2 := len(s1), len(s2)
	longest, comparisons := 0, 0

	for i := 0; i < size1; i++ {
		for j := 0; j < size2; j++ {
			length, m, n := 0, i, j

			for {
				if m < size1 && n < size2 {
					comparisons++
					if string(s1[m]) != string(s2[n]) {
						break
					}

					length++
					m++
					n++
				} else {
					break
				}
			}

			if longest < length {
				longest = length
			}
		}
	}

	return longest
}

// md5
func md5Hash(s string) string {
	hasher := md5.New()
	hasher.Write([]byte(s))
	return hex.EncodeToString(hasher.Sum(nil))
}

const (
	positive             = "(?i)article|body|content|entry|hentry|main|page|attachment|pagination|post|text|blog|story"
	negative             = "(?i)combx|comment|com-|contact|foot|footer|_nav|footnote|masthead|media|meta|outbrain|promo|related|scroll|shoutbox|sidebar|sponsor|shopping|tags|tool|widget"
	unlikelyCandidates   = "(?i)combx|comment|community|disqus|extra|foot|header|menu|remark|rss|shoutbox|sidebar|sponsor|ad-break|agegate|pagination|pager|popup|button"
	okMaybeItsACandidate = "(?i)and|article|body|column|main|shadow"
	divToPElements       = "(?i)<(a|blockquote|dl|div|img|ol|p|pre|table|ul)"
	replaceBrs           = "(?i)(<br[^>]*>[ \n\r\t]*){2,}"
	videos               = `(?i)http://(www|v\.)?(youtube|youku|tudou|ku6|qq|sina|sohu|vimeo)\.com`
)

type NodeScore struct {
	score float64
	node  *html.Node
}

type Readability struct {
	HTML       string
	URL        string
	candidates map[string]NodeScore
	title      string
	titleNode  *html.Node
	//	aNode      *html.Node
	maybeHTML string
}

func (r *Readability) initializeNode(n *html.Node) (nodeScore NodeScore) {
	contentScore := 0.0

	if n.Type == html.ElementNode {
		switch strings.ToLower(n.Data) {
		case "article":
			contentScore += 10
		case "section":
			contentScore += 8
		case "div":
			contentScore += 5
		case "pre", "td", "blockquote":
			contentScore += 3
		case "address", "ol", "ul", "dl", "dd", "dt", "li", "form":
			contentScore -= 3
		case "h1", "h2", "h3", "h4", "h5", "h6", "th":
			contentScore -= 5
		}
	}

	contentScore += r.classWeight(n)

	nodeScore.score = contentScore
	nodeScore.node = n

	return nodeScore
}

func (r *Readability) linkDensity(n *html.Node) float64 {
	nodes := getElementsByTagName(n, "a")
	textLength := len(getInnerText(n))
	if textLength == 0 {
		return 0.0
	}

	linkLength := 0
	for _, node := range nodes {
		linkLength += len(getInnerText(node))
	}

	return float64(linkLength) / float64(textLength)
}

func (r *Readability) classWeight(n *html.Node) float64 {
	weight := 0.0

	class := getAttribute(n, "class")
	if class != "" {
		if ok, e := regexp.MatchString(negative, class); ok && e == nil {
			weight -= 25
		}

		if ok, e := regexp.MatchString(positive, class); ok && e == nil {
			weight += 25
		}
	}

	id := getAttribute(n, "id")
	if id != "" {
		if ok, e := regexp.MatchString(negative, id); ok && e == nil {
			weight -= 25
		}

		if ok, e := regexp.MatchString(positive, id); ok && e == nil {
			weight += 25
		}
	}

	return weight
}

func (r *Readability) cleanByTagName(n *html.Node, tag string) {
	targetList := getElementsByTagName(n, tag)
	isEmbed := tag == "object" || tag == "embed"

	for _, target := range targetList {
		if isEmbed {
			attributeValues := ""
			for _, a := range target.Attr {
				attributeValues = attributeValues + a.Val + "|"
			}

			if ok, e := regexp.MatchString(videos, attributeValues); ok && e == nil {
				continue
			}

			if ok, e := regexp.MatchString(videos, getInnerHTML(target)); ok && e == nil {
				continue
			}
		}

		if target.Parent != nil {
			target.Parent.RemoveChild(target)
		}
	}
}

func (r *Readability) cleanConditionally(n *html.Node, tag string) {
	tagsList := getElementsByTagName(n, tag)

	for _, node := range tagsList {
		weight := r.classWeight(node)
		hashNode := md5Hash(getInnerHTML(node))

		contentScore := 0.0
		_, ok := r.candidates[hashNode]
		if ok {
			contentScore = r.candidates[hashNode].score
		}

		if weight+contentScore < 0 {
			if node.Parent != nil {
				node.Parent.RemoveChild(node)
			}
		} else {
			p := len(getElementsByTagName(node, "p"))
			img := len(getElementsByTagName(node, "img"))
			li := len(getElementsByTagName(node, "li")) - 100
			input := len(getElementsByTagName(node, "input"))
			embedCount := 0
			embeds := getElementsByTagName(node, "embed")
			for _, embed := range embeds {
				if ok, e := regexp.MatchString(videos, getAttribute(embed, "src")); !ok || e != nil {
					embedCount += 1
				}
			}

			linkDensity := r.linkDensity(node)
			contentLength := len(getInnerText(node))
			toRemove := false

			if img > p {
				toRemove = true
			} else if li > p && tag != "ul" && tag != "ol" {
				toRemove = true
			} else if input > int(math.Floor(float64(p)/3)) {
				toRemove = true
			} else if weight < 25 && linkDensity > 0.2 {
				toRemove = true
			} else if weight >= 25 && linkDensity > 0.5 {
				toRemove = true
			} else if (embedCount == 1 && contentLength < 35) || embedCount > 1 {
				toRemove = true
			}

			if toRemove && node.Parent != nil {
				node.Parent.RemoveChild(node)
			}
		}
	}
}

func (r *Readability) cleanContent(n *html.Node) {
	r.cleanByTagName(n, "h1")
	r.cleanConditionally(n, "form")
	r.cleanByTagName(n, "h2")
	r.cleanByTagName(n, "object")
	r.cleanByTagName(n, "iframe")

	r.cleanConditionally(n, "table")
	r.cleanConditionally(n, "ul")
	r.cleanConditionally(n, "div")

	r.fixImagesPath(n)
}

func (r *Readability) stripUnlikely(n *html.Node) {
	if n.Type == html.ElementNode && strings.ToLower(n.Data) == "div" {
		unlikelyMatchString := getAttribute(n, "class") + getAttribute(n, "id")
		m, _ := regexp.MatchString(unlikelyCandidates, unlikelyMatchString)
		_m, _ := regexp.MatchString(okMaybeItsACandidate, unlikelyMatchString)
		if m && !_m && strings.ToLower(n.Data) != "body" && n.Parent != nil {
			n.Parent.RemoveChild(n)
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		r.stripUnlikely(c)
	}
}

func (r *Readability) fixImagesPath(n *html.Node) {
	if n.Type == html.ElementNode && strings.ToLower(n.Data) == "img" {
		src := getAttribute(n, "src")
		if src == "" && n.Parent != nil {
			n.Parent.RemoveChild(n)
		} else {
			src = rel2abs(src, r.URL)
			setAttribute(n, "src", src)
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		r.fixImagesPath(c)
	}
}

func (r *Readability) divToPElements(n *html.Node) {
	if n.Type == html.ElementNode && strings.ToLower(n.Data) == "div" {
		m, ok := regexp.MatchString("(?is)<(a|blockquote|dl|div|img|ol|p|pre|table|ul)", getInnerHTML(n))
		if !m || ok != nil {
			n.Data = "p"
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		r.divToPElements(c)
	}
}

func (r *Readability) pToBRElements(n *html.Node) {
	if n.Type == html.ElementNode && strings.ToLower(n.Data) == "p" {
		n.Data = "br"
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		r.pToBRElements(c)
	}
}

func (r *Readability) GetContent() string {
	re, _ := regexp.Compile(replaceBrs)
	data := re.ReplaceAllString(r.HTML, "</p><p>")

	re, _ = regexp.Compile(`(?i)<style[\S\s]+?</style>`)
	data = re.ReplaceAllString(data, "")

	re, _ = regexp.Compile(`(?i)<script[\S\s]+?</script>`)
	data = re.ReplaceAllString(data, "")
	// data = strings.Replace(data, `&lt;&gt;`, "", -1)
	//fmt.Println(data)

	root, err := html.Parse(strings.NewReader(data))
	if err != nil {
		//	fmt.Println("html parse error")
		return ""
	}

	r.stripUnlikely(root)
	r.divToPElements(root)

	var nodes []*html.Node
	var nodesToScore func(*html.Node)
	nodesToScore = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.ToLower(n.Data) == "p" {
			nodes = append(nodes, n)
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			nodesToScore(c)
		}
	}
	nodesToScore(root)

	var parentNode *html.Node
	var grandParentNode *html.Node
	for _, n := range nodes {
		if n.Parent != nil {
			parentNode = n.Parent
		}

		if parentNode.Parent != nil {
			grandParentNode = parentNode.Parent
		}

		innerText := getInnerText(n)
		if parentNode == nil || len(innerText) < 20 {
			continue
		}

		parentHash := md5Hash(getInnerHTML(parentNode))
		grandParentHash := md5Hash(getInnerHTML(grandParentNode))
		_, ok := r.candidates[parentHash]
		if !ok {
			r.candidates[parentHash] = r.initializeNode(parentNode)
		}

		if grandParentNode != nil {
			_, ok := r.candidates[grandParentHash]
			if !ok {
				r.candidates[grandParentHash] = r.initializeNode(grandParentNode)
			}
		}

		contentScore := 1.0
		contentScore += float64(strings.Count(innerText, ","))
		contentScore += float64(strings.Count(innerText, "，"))
		contentScore += math.Min(math.Floor(float64(len(innerText))/100), 3)
		nodeScore := r.candidates[parentHash]
		nodeScore.score += contentScore
		r.candidates[parentHash] = nodeScore

		if grandParentNode != nil {
			nodeScore := r.candidates[grandParentHash]
			nodeScore.score += contentScore / 2.0
			r.candidates[grandParentHash] = nodeScore
		}
	}

	var topCandidate NodeScore
	//topCandidate.score = 0
	for _, nodeScore := range r.candidates {
		nodeScore.score = nodeScore.score * (1 - r.linkDensity(nodeScore.node))
		if nodeScore.score > topCandidate.score {
			topCandidate = nodeScore
		}
	}

	content := ""
	if topCandidate.score > 0.0 {
		r.cleanContent(topCandidate.node)

		//		beego.Info("topCandidate.node: ", getInnerHTML(topCandidate.node))

		//		// 去除含有链接的节点，主要针对小说正文优化
		//		candidates := map[string]NodeScore{}
		//		aNodes := getElementsByTagName(topCandidate.node, "a")
		//		var pNode *html.Node
		//		for _, n := range aNodes {
		//			if n.Parent != nil {
		//				pNode = n.Parent
		//			}

		//			innerText := getInnerText(n)
		//			if pNode == nil || len(innerText) < 20 {
		//				continue
		//			}

		//			pHash := md5Hash(getInnerHTML(pNode))
		//			_, ok := candidates[pHash]
		//			if !ok {
		//				nodeScore := NodeScore{}
		//				nodeScore.node = pNode
		//				nodeScore.score = 1.0
		//				candidates[pHash] = nodeScore
		//			} else {
		//				nodeScore := candidates[pHash]
		//				nodeScore.score = nodeScore.score + 1.0
		//				candidates[pHash] = nodeScore
		//			}
		//		}

		//		for _, score := range candidates {
		//			if score.score > 1 {
		//				score.node.Parent.RemoveChild(score.node)
		//			}
		//		}

		//		beego.Info("topCandidate.node: ", getInnerHTML(topCandidate.node))

		content = getInnerHTML(topCandidate.node)
	}
	//fmt.Println("content: ", content)
	if len(content) == 0 {
		content = r._getContent()
		//fmt.Println("_getContent: ", content)
	}

	return content
}

// 获取标题和第一个链接之间的正文
func (r *Readability) _getContent() string {
	if len(r.title) == 0 {
		r.GetTitle()
	}

	if r.maybeHTML == "" {
		return ""
	}

	//fmt.Println("r.titleNode: ", getInnerHTML(r.titleNode))

	//return r.maybeHTML
	//fmt.Println("r.maybeHTML: ", r.maybeHTML)

	root, err := html.Parse(strings.NewReader(r.maybeHTML))
	if err != nil {
		return ""
	}

	isEnd := false
	var hasContent bool
	var data string
	var getContent func(*html.Node)
	getContent = func(n *html.Node) {
		ok := false
		content := getInnerHTML(n)
		r := strings.NewReplacer("<br>", "\r\n", "<br/>", "\r\n", "<br />", "\r\n")
		content = r.Replace(content)
		ok, _ = regexp.MatchString("\\<[\\S\\s]+?\\>", content)

		isA := false
		if !ok && n.Type == html.ElementNode && strings.ToLower(n.Data) == "a" {
			isA = true
		}

		if !ok && hasContent && isA {
			isEnd = true
			return
		}

		isContent := false
		if !ok && n.Type == html.ElementNode {
			e := strings.ToLower(n.Data)
			if e == "div" || e == "p" {
				isContent = true
			}
		}
		if isContent && len(getInnerText(n)) > 5 {
			hasContent = true
		}
		if isContent {
			data += getInnerHTML(n)
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if isEnd {
				break
			}
			getContent(c)
		}
	}

	getContent(root)

	return data
}

// 通过求最大公共子串来获取标题
func (r *Readability) GetTitle() string {
	if len(r.title) > 0 {
		return r.title
	}

	re, err := regexp.Compile(`(?i)<title>(.*?)</title>`)
	if err != nil {
		return ""
	}

	ar := re.FindStringSubmatch(r.HTML)
	title := ""
	if len(ar) > 0 {
		title = strings.ToLower(ar[1])
	}

	if title != "" {
		i := len(title)
		root, err := html.Parse(strings.NewReader(r.HTML))
		if err != nil {
			return title
		}

		j := 0
		var getTitle func(*html.Node, string)
		getTitle = func(n *html.Node, t string) {
			if n.Type == html.TextNode {
				str := n.Data
				x := len(str)
				if x > int(float64(i)*0.3) && x < i {
					m := longestCommonSubstring(t, str)
					if m >= j {
						title = str
						r.titleNode = n.Parent
						j = m
					}
				}
			}

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				getTitle(c, t)
			}
		}
		getTitle(root, title)

		hasBegin := false
		var getHTML func(*html.Node, string)
		getHTML = func(n *html.Node, t string) {
			if n.Type == html.TextNode {
				str := n.Data
				x := len(str)
				if x > int(float64(i)*0.3) && x < i {
					m := longestCommonSubstring(t, str)
					if m >= j {
						hasBegin = true
					}
				}
			}

			if hasBegin {
				r.maybeHTML += getInnerHTML(n.Parent)
			}

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				getHTML(c, t)
			}
		}
		getHTML(root, title)
	}

	r.title = title
	return r.title
}

func NewReadability(html, url string) *Readability {
	r := &Readability{HTML: html, URL: url, candidates: make(map[string]NodeScore)}
	return r
}
