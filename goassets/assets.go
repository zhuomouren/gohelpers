// 变量程序生成，增加 Get 方法获取数据
// 生成的文件添加解码函数
// 打包资源文件
package goassets

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"go/format"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zhuomouren/gohelpers/gofile"
	"github.com/zhuomouren/gohelpers/gostring"
)

type goassets struct {
	mux   sync.RWMutex
	cache map[string]*GoAssets
}

func (this *goassets) add(path string, goAssets *GoAssets) (added bool) {
	this.mux.Lock()
	defer this.mux.Unlock()
	if _, ok := this.cache[path]; !ok {
		this.cache[path] = goAssets
		added = true
	}
	return
}

func (this *goassets) get(path string) (goAssets *GoAssets, ok bool) {
	this.mux.RLock()
	defer this.mux.RUnlock()
	goAssets, ok = this.cache[path]
	return
}

var (
	assets = &goassets{cache: make(map[string]*GoAssets)}
)

type GoAssets struct {
	packageName    string // 包名
	assetsPath     string // 资源路径
	assetsSavePath string
	root           string
	exts           []string
	assets         []Asset
}

type Asset struct {
	Path    string
	Mode    os.FileMode
	ModTime time.Time
	Data    template.HTML
	// Info    os.FileInfo
}

//
func NewGoAssets(assetsPath string) *GoAssets {
	if goAssets, ok := assets.get(assetsPath); ok {
		return goAssets
	}

	goAssets := &GoAssets{
		assetsPath: assetsPath,
		assets:     []Asset{},
	}

	if ok := assets.add(assetsPath, goAssets); !ok {
		return nil
	}

	return goAssets
}

func (this *GoAssets) AddAsset(path string, modTime time.Time, data string) {
	asset := Asset{
		Path:    path,
		ModTime: modTime,
		Data:    template.HTML(data),
	}

	this.assets = append(this.assets, asset)
}

func (this *GoAssets) SetExts(exts []string) {
	this.exts = exts
}

func (this *GoAssets) AddExts(ext string) {
	this.exts = append(this.exts, ext)
}

func (this *GoAssets) RemoveExts(ext string) {
	this.exts = gostring.Helper.RemoveSlice(this.exts, ext)
}

func (this *GoAssets) Build(root, packageName, savePath string) error {
	this.root = cleanPath(root)
	this.packageName = packageName
	this.assetsSavePath = savePath

	if this.root == "" || this.packageName == "" || this.assetsSavePath == "" {
		return errors.New("root or packageName or savePath cannot be empty.")
	}

	var err error

	t := template.New("asset.tpl")
	t, err = t.Parse(assetTmpl)
	if err != nil {
		return err
	}

	err = this.parse()
	if err != nil {
		return err
	}

	w := &bytes.Buffer{}
	t.Execute(w, map[string]interface{}{
		"PackageName": this.packageName,
		"Assets":      this.assets,
	})

	source, err := format.Source(w.Bytes())
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(this.assetsSavePath, source, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func (this *GoAssets) parse() error {
	var err error

	var files []string
	path := cleanJoinPath(this.root, this.assetsPath)
	if len(this.exts) > 0 {
		_, files, err = gofile.FileHelper.AllowedFiles(path, this.exts)
	} else {
		_, files, err = gofile.FileHelper.Files(path)
	}

	if err != nil {
		return err
	}

	for _, file := range files {
		if strings.HasSuffix(file, ".DS_Store") {
			continue
		}

		file = cleanPath(file)

		asset, err := this.readFile(file)
		if err != nil {
			return err
		}

		asset.Path = strings.TrimPrefix(file, this.root)
		this.assets = append(this.assets, asset)
	}

	return nil
}

func (this *GoAssets) readFile(assetFile string) (Asset, error) {
	asset := Asset{}

	f, err := os.Open(assetFile)
	if err != nil {
		return asset, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return asset, err
	}

	// asset.Info = fi
	asset.Mode = fi.Mode()
	asset.ModTime = fi.ModTime()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return asset, err
	}

	var buf bytes.Buffer
	gw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return asset, err
	}
	if _, err := gw.Write(data); err != nil {
		return asset, err
	}
	if err := gw.Close(); err != nil {
		return asset, err
	}

	var b bytes.Buffer
	b64 := base64.NewEncoder(base64.StdEncoding, &b)
	b64.Write(buf.Bytes())
	b64.Close()
	res := "\n"
	// 每行 80 个字符
	chunk := make([]byte, 80)
	for n, _ := b.Read(chunk); n > 0; n, _ = b.Read(chunk) {
		res += string(chunk[0:n]) + "\n"
	}

	asset.Data = template.HTML(res)

	return asset, nil
}

func cleanJoinPath(paths ...string) string {
	return cleanPath(filepath.Join(paths...))
}
func cleanPath(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}
