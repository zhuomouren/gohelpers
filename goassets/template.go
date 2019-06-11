package goassets

var assetTmpl = `
package {{.PackageName}}

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type myasset struct {
	path    string
	modTime time.Time
	data    string
}

var assets = []myasset{
	{{range $asset := .Assets}}
	myasset{
		path:"{{$asset.Path}}",
		modTime: time.Unix({{$asset.ModTime.Unix}}, {{$asset.ModTime.UnixNano}}),
		data: ` + "`{{$asset.Data}}`" + `,
	},
	{{end}}
}

func AssetsPaths() []string {
	var assetsPaths []string

	for _, asset := range assets {
		assetsPaths = append(assetsPaths, asset.path)
	}

	return assetsPaths
}

func GetAsset(name string) ([]byte, error) {
	name = filepath.ToSlash(name)
	for _, asset := range assets {
		if strings.EqualFold(name, asset.path) {
			data, err := decodeAsset(asset.data)
			if err != nil {
				return nil, err
			}

			return data, nil
		}
	}

	return nil, os.ErrNotExist
}

func readAsset(name string) (myasset, error) {
	name = filepath.ToSlash(name)
	for _, asset := range assets {
		if strings.EqualFold(name, asset.path) {
			return asset, nil
		}
	}

	return myasset{}, os.ErrNotExist
}

func decodeAsset(assetData string) ([]byte, error) {
	b64 := base64.NewDecoder(base64.StdEncoding, bytes.NewBufferString(assetData))
	gr, err := gzip.NewReader(b64)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(gr)
	if err != nil {
		return nil, err
	}
	return data, nil
}

type AssetFileInfo struct {
	name      string
	isDir     bool
	len       int64
	timestamp time.Time
}

func newAssetFileInfo(name string, len int64, timestamp time.Time) AssetFileInfo {
	return AssetFileInfo{
		name:      name,
		isDir:     false,
		len:       len,
		timestamp: timestamp,
	}
}

func (this *AssetFileInfo) Name() string {
	return this.name
}

func (this *AssetFileInfo) Mode() os.FileMode {
	mode := os.FileMode(0644)
	if this.isDir {
		return mode | os.ModeDir
	}
	return mode
}

func (this *AssetFileInfo) ModTime() time.Time {
	return this.timestamp
}

func (this *AssetFileInfo) Size() int64 {
	return this.len
}

func (this *AssetFileInfo) IsDir() bool {
	return this.Mode().IsDir()
}

func (this *AssetFileInfo) Sys() interface{} {
	return nil
}

type AssetFile struct {
	*bytes.Reader
	io.Closer
	AssetFileInfo
}

func NewAssetFile(name string, content []byte, timestamp time.Time) *AssetFile {
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	return &AssetFile{
		bytes.NewReader(content),
		ioutil.NopCloser(nil),
		newAssetFileInfo(name, int64(len(content)), timestamp)}
}

func (this *AssetFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, errors.New("not a directory")
}

func (this *AssetFile) Size() int64 {
	return this.AssetFileInfo.Size()
}

func (this *AssetFile) Stat() (os.FileInfo, error) {
	return this, nil
}

type AssetFS struct {
	path  string
	files map[string]*AssetFile
}

func NewAssetFS(path string) *AssetFS {
	assetFS := &AssetFS{
		path: path,
	}
	assetFS.files = make(map[string]*AssetFile, 0)

	return assetFS
}

func (this *AssetFS) Open(name string) (http.File, error) {
	if f, ok := this.files[name]; ok {
		return f, nil
	}

	key := strings.TrimPrefix(name, this.path)
	if len(key) > 0 && key[0] == '/' {
		key = key[1:]
	}
	asset, err := readAsset(key)
	if err != nil {
		return nil, err
	}

	data, err := decodeAsset(asset.data)
	if err != nil {
		return nil, err
	}

	assetFile := NewAssetFile(key, data, asset.modTime)
	this.files[name] = assetFile

	return assetFile, nil
}

func AssetsDir(path string) *AssetFS {
	return NewAssetFS(path)
}
`
