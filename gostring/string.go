package gostring

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
)

type GoString struct{}

var Helper = &GoString{}

func (this *GoString) MD5(str string) string {
	data := []byte(str)
	md5Ctx := md5.New()
	md5Ctx.Write(data)
	cipherStr := md5Ctx.Sum(nil)
	return hex.EncodeToString(cipherStr)
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

func (this *GoString) InSlice(value string, values []string) bool {
	exists := false

	for _, val := range values {
		if strings.EqualFold(val, value) {
			exists = true
			break
		}
	}

	return exists
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
