package goassets

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/zhuomouren/gohelpers/gostring"
)

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
	path        string
	allowedExts []string
	blockedExts []string
	files       map[string]*AssetFile
	mux         *sync.Mutex
}

func NewAssetFS(path string, allowedExts, blockedExts []string) *AssetFS {
	assetFS := &AssetFS{
		path:        path,
		allowedExts: allowedExts,
		blockedExts: blockedExts,
		mux:         &sync.Mutex{},
	}
	assetFS.files = make(map[string]*AssetFile, 0)

	return assetFS
}

func (this *AssetFS) Open(name string) (http.File, error) {
	this.mux.Lock()
	defer this.mux.Unlock()

	key := cleanJoinPath(this.path, name)
	ext := filepath.Ext(key)
	if gostring.Helper.InSlice(ext, this.blockedExts) {
		return nil, os.ErrNotExist
	}

	if len(this.allowedExts) > 0 && !gostring.Helper.InSlice(ext, this.allowedExts) {
		return nil, os.ErrNotExist
	}

	if f, ok := this.files[key]; ok {
		return f, nil
	}

	asset, err := Assets.ReadAsset(key)
	if err != nil {
		return nil, err
	}

	data := []byte(asset.Data)
	assetFile := NewAssetFile(key, data, asset.ModTime)
	this.files[name] = assetFile

	return assetFile, nil
}

func AssetsDir(path string) *AssetFS {
	return NewAssetFS(path, []string{}, []string{})
}

func AssetsDirAllowedExts(path string, allowedExts []string) *AssetFS {
	return NewAssetFS(path, allowedExts, []string{})
}

func AssetsDirBlockedExts(path string, blockedExts []string) *AssetFS {
	return NewAssetFS(path, []string{}, blockedExts)
}
