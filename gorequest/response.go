package gorequest

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/kennygrant/sanitize"
)

type GoResponse struct {
	StatusCode int
	cost       time.Duration
	data       []byte
	request    *GoRequest
	headers    *http.Header
}

func (this *GoResponse) Request() *GoRequest {
	return this.request
}

func (this *GoResponse) Bytes() []byte {
	return this.data
}

func (this *GoResponse) String() string {
	return string(this.data)
}

func (this *GoResponse) ToJSON(v interface{}) error {
	return json.Unmarshal(this.data, v)
}

func (this *GoResponse) ToXML(v interface{}) error {
	return xml.Unmarshal(this.data, v)
}

func (this *GoResponse) Save(fileName string) error {
	return ioutil.WriteFile(fileName, this.data, 0644)
}

func (this *GoResponse) FileName() string {
	_, params, err := mime.ParseMediaType(this.headers.Get("Content-Disposition"))
	if fName, ok := params["filename"]; ok && err == nil {
		return SanitizeFileName(fName)
	}
	if this.request.URL.RawQuery != "" {
		return SanitizeFileName(fmt.Sprintf("%s_%s", this.request.URL.Path, this.request.URL.RawQuery))
	}
	return SanitizeFileName(strings.TrimPrefix(this.request.URL.Path, "/"))
}

// SanitizeFileName replaces dangerous characters in a string
// so the return value can be used as a safe file name.
func SanitizeFileName(fileName string) string {
	ext := filepath.Ext(fileName)
	cleanExt := sanitize.BaseName(ext)
	if cleanExt == "" {
		cleanExt = ".unknown"
	}
	return strings.Replace(fmt.Sprintf(
		"%s.%s",
		sanitize.BaseName(fileName[:len(fileName)-len(ext)]),
		cleanExt[1:],
	), "-", "_", -1)
}
