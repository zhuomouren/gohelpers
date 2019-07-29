package gonet

import (
	"errors"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type GoURL struct{}

var URLHelper = &GoURL{}

// 是否 http:// 或 https:// 开头
func (this *GoURL) HasPrefix(url string) bool {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return true
	}

	return false
}

// 移除 URL 的 HTTP 或 HTTPS 前缀
func (this *GoURL) RemoveHTTPPrefix(url string) string {
	httpPrefixRE := regexp.MustCompile(`^https?://`)
	match := httpPrefixRE.ReplaceAllString(url, "")

	return match
}

// 美化 url
func (this *GoURL) Clean(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	if !u.IsAbs() {
		return uri
	}

	scheme := u.Scheme
	str := strings.Replace(uri, scheme+"://", "", -1)

	return scheme + "://" + path.Clean(str)
}

func (this *GoURL) Join(str ...string) string {
	return this.Clean(strings.Join(str, "/"))
}

// 相对路径转绝对路径
func (this *GoURL) AbsoluteURL(rel, base string) (string, error) {
	if rel == "" || strings.HasPrefix(rel, "#") {
		return "", errors.New("Can't start with #")
	}

	u, err := url.Parse(rel)
	if err != nil {
		return "", err
	}

	if u.IsAbs() {
		return rel, nil
	}

	if rel[0] == '#' || rel[0] == '?' {
		return base + rel, nil
	}

	u, err = url.Parse(base)
	if err != nil {
		return "", err
	}
	path := u.Path
	if filepath.Ext(u.Path) != "" {
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

	return u.Scheme + "://" + strings.Replace(strings.TrimLeft(strings.Join(dst, "/"), "/"), "//", "/", -1), nil
}
