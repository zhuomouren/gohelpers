package gorequest

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"
)

// Resp represents a request with it's response
type GoResponse struct {
	request                  *http.Request
	response                 *http.Response
	client                   *http.Client
	cost                     time.Duration
	requestBody              []byte
	responseBody             []byte
	downloadProgressInterval time.Duration
	downloadProgress         func(int64, int64)
	err                      error // delayed error
}

// Request returns *http.Request
func (this *GoResponse) Request() *http.Request {
	return this.request
}

// Response returns *http.Response
func (this *GoResponse) Response() *http.Response {
	return this.response
}

// Bytes returns response body as []byte
func (this *GoResponse) Bytes() []byte {
	data, _ := this.ToBytes()
	return data
}

// ToBytes returns response body as []byte,
// return error if error happend when reading
// the response body
func (this *GoResponse) ToBytes() ([]byte, error) {
	if this.err != nil {
		return nil, this.err
	}
	if this.responseBody != nil {
		return this.responseBody, nil
	}
	defer this.response.Body.Close()
	respBody, err := ioutil.ReadAll(this.response.Body)
	if err != nil {
		this.err = err
		return nil, err
	}
	this.responseBody = respBody
	return this.responseBody, nil
}

// String returns response body as string
func (this *GoResponse) String() string {
	data, _ := this.ToBytes()
	return string(data)
}

// ToString returns response body as string,
// return error if error happend when reading
// the response body
func (this *GoResponse) ToString() (string, error) {
	data, err := this.ToBytes()
	return string(data), err
}

// ToJSON convert json response body to struct or map
func (this *GoResponse) ToJSON(v interface{}) error {
	data, err := this.ToBytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// ToXML convert xml response body to struct or map
func (this *GoResponse) ToXML(v interface{}) error {
	data, err := this.ToBytes()
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, v)
}

// ToFile download the response body to file with optional download callback
func (this *GoResponse) ToFile(name string) error {
	//TODO set name to the suffix of url path if name == ""
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	defer file.Close()

	if this.responseBody != nil {
		_, err = file.Write(this.responseBody)
		return err
	}

	if this.downloadProgress != nil && this.response.ContentLength > 0 {
		return this.download(file)
	}

	defer this.response.Body.Close()
	_, err = io.Copy(file, this.response.Body)
	return err
}

func (this *GoResponse) download(file *os.File) error {
	p := make([]byte, 1024)
	b := this.response.Body
	defer b.Close()
	total := this.response.ContentLength
	var current int64
	var lastTime time.Time

	defer func() {
		this.downloadProgress(current, total)
	}()

	for {
		l, err := b.Read(p)
		if l > 0 {
			_, _err := file.Write(p[:l])
			if _err != nil {
				return _err
			}
			current += int64(l)
			if now := time.Now(); now.Sub(lastTime) > this.downloadProgressInterval {
				lastTime = now
				this.downloadProgress(current, total)
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

var regNewline = regexp.MustCompile(`\n|\r`)

func (this *GoResponse) autoFormat(s fmt.State) {
	req := this.request
	fmt.Fprint(s, req.Method, " ", req.URL.String(), " ", this.cost)

	// test if it is should be outputed pretty
	var pretty bool
	var parts []string
	addPart := func(part string) {
		if part == "" {
			return
		}
		parts = append(parts, part)
		if !pretty && regNewline.MatchString(part) {
			pretty = true
		}
	}
	addPart(string(this.requestBody))
	addPart(this.String())

	for _, part := range parts {
		if pretty {
			fmt.Fprint(s, "\n")
		}
		fmt.Fprint(s, " ", part)
	}
}

// Format fort the response
func (this *GoResponse) Format(s fmt.State, verb rune) {
	if this == nil || this.request == nil {
		return
	}

	this.autoFormat(s)
}
