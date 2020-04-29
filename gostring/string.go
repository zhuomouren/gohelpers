package gostring

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/catinello/base62"
	"github.com/google/uuid"
	"github.com/sony/sonyflake"
)

type GoString struct{}

var Helper = &GoString{}

func (this *GoString) UUID() string {
	return uuid.New().String()
}

var flake = NewSonyflake()

// fix no private ip address error
func NewSonyflake() *sonyflake.Sonyflake {
	st := sonyflake.Settings{}
	st.MachineID = func() (uint16, error) {
		ip, err := privateIPv4()
		if err != nil {
			ip = net.IP([]byte{192, 168, 1, 1})
		}

		return uint16(ip[2])<<8 + uint16(ip[3]), nil
	}

	return sonyflake.NewSonyflake(st)
}

func privateIPv4() (net.IP, error) {
	as, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, a := range as {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}

		ip := ipnet.IP.To4()
		if isPrivateIPv4(ip) {
			return ip, nil
		}
	}
	return nil, errors.New("no private ip address")
}

func isPrivateIPv4(ip net.IP) bool {
	return ip != nil &&
		(ip[0] == 10 || ip[0] == 172 && (ip[1] >= 16 && ip[1] < 32) || ip[0] == 192 && ip[1] == 168)
}

func (this *GoString) ShortUUID() string {
	// flake := sonyflake.NewSonyflake(sonyflake.Settings{})
	if flake == nil {
		log.Println("sonyflake is nil")
		return this.UUID()
	}

	id, err := flake.NextID()
	if err != nil {
		log.Printf("flake.NextID() failed with %s\n", err)
		return this.UUID()
	}

	// 16 进制
	// return fmt.Sprintf("%x", id)

	// 使用 base62 编码更短
	return base62.Encode(int(id))
}

// 单位转换
func (this *GoString) HumanBytes(size int64) string {
	suffix := ""
	i := size
	if size > (1 << 30) {
		suffix = "G"
		i = size / (1 << 30)
	} else if size > (1 << 20) {
		suffix = "M"
		i = size / (1 << 20)
	} else if size > (1 << 10) {
		suffix = "K"
		i = size / (1 << 10)
	}
	return fmt.Sprintf("%d%s", i, suffix)
}

// md5编码
func (this *GoString) MD5(str string) string {
	data := []byte(str)
	md5Ctx := md5.New()
	md5Ctx.Write(data)
	cipherStr := md5Ctx.Sum(nil)
	return hex.EncodeToString(cipherStr)
}

// 去除头尾斜杠 /
func (this *GoString) TrimSlash(str string) string {
	return strings.Trim(str, "/")
}

// 去除开头斜杠 /
func (this *GoString) TrimLeftSlash(str string) string {
	return strings.TrimLeft(str, "/")
}

// 去除结尾斜杠 /
func (this *GoString) TrimRightSlash(str string) string {
	return strings.TrimRight(str, "/")
}

// 以斜杠开头,结尾没有斜杠
func (this *GoString) LeftSlash(str string) string {
	return "/" + strings.Trim(str, "/")
}

// 以斜杠结尾, 开头没有斜杠
func (this *GoString) RightSlash(str string) string {
	return strings.Trim(str, "/") + "/"
}

// 根据大写字母分隔字符串
func (this *GoString) SplitUpper(data string) []string {
	re, _ := regexp.Compile("[A-Z]")
	rep := re.ReplaceAllStringFunc(data, func(str string) string {
		return " " + str
	})

	return strings.Fields(rep)
}

// 随机字符串
func (this *GoString) GetRandomString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz!@#$%^&()"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func (this *GoString) ToBool(value string) (bool, error) {
	if strings.EqualFold(value, "on") {
		return true, nil
	}

	return strconv.ParseBool(value)
}

func (this *GoString) ToInt(value string) (int, error) {
	return strconv.Atoi(value)
}

func (this *GoString) ToUint(value string) (uint, error) {
	v, err := strconv.ParseUint(value, 10, 32)
	return uint(v), err
}

func (this *GoString) ToInt64(value string) (int64, error) {
	return strconv.ParseInt(value, 10, 64)
}

func (this *GoString) ToUint64(value string) (uint64, error) {
	return strconv.ParseUint(value, 10, 64)
}

// 不区分大小写
func (this *GoString) InSlice(value string, values []string) bool {
	for _, val := range values {
		if strings.EqualFold(val, value) {
			return true
		}
	}

	return false
}

// 区分大小写
func (this *GoString) SliceContains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func (this *GoString) RemoveSlice(values []string, value string) []string {
	index := -1
	for i, val := range values {
		if strings.Contains(val, value) {
			index = i
			break
		}
	}

	if index >= 0 {
		ret := make([]string, len(values))
		copy(ret, values)
		ret = append(ret[:index], ret[index+1:]...)
		return ret
	}

	return values
}

// 去重
func (this *GoString) RemoveDuplicate(slis *[]string) {
	found := make(map[string]bool)
	j := 0
	for i, val := range *slis {
		if _, ok := found[val]; !ok {
			found[val] = true
			(*slis)[j] = (*slis)[i]
			j++
		}
	}

	*slis = (*slis)[:j]
}

func (this *GoString) Sub(html, begin, end string) string {
	if html != "" && begin != "" && end != "" {
		s := strings.Split(html, begin)
		if len(s) > 1 {
			ss := strings.Split(s[1], end)
			if len(ss) > 1 {
				html = ss[0]
			}
		}
	}

	return html
}

// {数字} => ^[0-9]+$
// [数字] => (^[0-9]+$)
// {内容} => .*?
// [内容] => (.*?)
func (this *GoString) DeepProcessingRegex(regex string) string {
	regex = strings.Replace(regex, "{数字}", "[0-9]+", -1)
	regex = strings.Replace(regex, "[数字]", "([0-9]+)", -1)
	regex = strings.Replace(regex, "{字母}", "[A-Za-z]+", -1)
	regex = strings.Replace(regex, "[字母]", "([A-Za-z]+)", -1)
	regex = strings.Replace(regex, "{字母数字}", "[0-9A-Za-z]+", -1)
	regex = strings.Replace(regex, "[字母数字]", "([0-9A-Za-z]+)", -1)
	regex = strings.Replace(regex, "{内容}", ".*?", -1)
	regex = strings.Replace(regex, "[内容]", "(.*?)", -1)
	regex = strings.Replace(regex, "{URL}", "[^/:]+", -1)
	regex = strings.Replace(regex, "[URL]", "([^/:]+)", -1)

	return regex
}

// 所有匹配
func (this *GoString) RegexpAllMatch(regex, data string) (matches []string) {
	if regex == "" || data == "" {
		return matches
	}

	regex = this.DeepProcessingRegex(regex)

	re := regexp.MustCompile(`(?i)` + regex)
	results := re.FindAllStringSubmatch(data, -1)
	for _, match := range results {
		matches = append(matches, match[1])
	}

	return matches
}

// 一个匹配
func (this *GoString) RegexpOneMatch(regex, data string) string {
	if regex == "" || data == "" {
		return ""
	}

	regex = this.DeepProcessingRegex(regex)

	re, err := regexp.Compile(`(?i)` + regex)
	if err != nil {
		return ""
	}

	ar := re.FindStringSubmatch(data)
	if len(ar) > 1 {
		return strings.TrimSpace(ar[1])
	}

	return ""
}

// 是否精确匹配
func (this *GoString) IsExactMatch(regex, data string) bool {
	re, err := regexp.Compile(`(?i)` + regex)
	if err != nil {
		return false
	}

	str := re.FindString(data)
	return strings.EqualFold(str, data)
}

// 是否存在匹配
func (this *GoString) IsMatch(regex, data string) bool {
	if m, _ := regexp.MatchString(regex, data); !m {
		return false
	}

	return true
}

// 随机字符串
func (this *GoString) Random(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}
