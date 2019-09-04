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

// 美化 url. 如果没有 scheme，将以斜杠开头
func (this *GoURL) Clean(uri string) string {
	if !this.HasPrefix(uri) {
		return "/" + strings.TrimLeft(path.Clean(uri), "/")
	}

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

// isDomainName checks if a string is a presentation-format domain name
// (currently restricted to hostname-compatible "preferred name" LDH labels and
// SRV-like "underscore labels"; see golang.org/issue/12421).
func (this *GoURL) IsDomainName(s string) bool {
	// See RFC 1035, RFC 3696.
	// Presentation format has dots before every label except the first, and the
	// terminal empty label is optional here because we assume fully-qualified
	// (absolute) input. We must therefore reserve space for the first and last
	// labels' length octets in wire format, where they are necessary and the
	// maximum total length is 255.
	// So our _effective_ maximum is 253, but 254 is not rejected if the last
	// character is a dot.
	l := len(s)
	if l == 0 || l > 254 || l == 254 && s[l-1] != '.' {
		return false
	}

	last := byte('.')
	nonNumeric := false // true once we've seen a letter or hyphen
	partlen := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		default:
			return false
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c == '_':
			nonNumeric = true
			partlen++
		case '0' <= c && c <= '9':
			// fine
			partlen++
		case c == '-':
			// Byte before dash cannot be dot.
			if last == '.' {
				return false
			}
			partlen++
			nonNumeric = true
		case c == '.':
			// Byte before dot cannot be dot, dash.
			if last == '.' || last == '-' {
				return false
			}
			if partlen > 63 || partlen == 0 {
				return false
			}
			partlen = 0
		}
		last = c
	}
	if last == '-' || partlen > 63 {
		return false
	}

	return nonNumeric
}
