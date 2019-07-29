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
	"time"

	"github.com/zhuomouren/gohelpers/gofile"
	"github.com/zhuomouren/gohelpers/gostring"
)

var (
	Assets = NewGoAssets()
)

type GoAssets struct {
	packageName    string   // 包名
	assetPaths     []string // 资源路径
	assetsSavePath string
	root           string
	exts           []string
	assets         []Asset
}

type Asset struct {
	Path    string
	Mode    os.FileMode
	ModTime time.Time
	Data    string
	// Info    os.FileInfo
}

func (this *Asset) GetRawData() ([]byte, error) {
	return decodeAsset(this.Data)
}

//
func NewGoAssets() *GoAssets {
	goAssets := &GoAssets{
		assets: []Asset{},
	}

	return goAssets
}

func (this *GoAssets) Exists() bool {
	return len(this.assets) > 0
}

func (this *GoAssets) GetAsset(name string) ([]byte, error) {
	name = filepath.ToSlash(name)
	for _, asset := range this.assets {
		if strings.EqualFold(name, asset.Path) {
			data, err := this.DecodeAsset(asset.Data)
			if err != nil {
				return nil, err
			}

			return data, nil
		}
	}

	return nil, os.ErrNotExist
}

func (this *GoAssets) ReadAsset(name string) (Asset, error) {
	name = filepath.ToSlash(name)
	for _, asset := range this.assets {
		if strings.EqualFold(name, asset.Path) {
			data, err := this.DecodeAsset(asset.Data)
			if err != nil {
				return Asset{}, err
			}
			asset.Data = string(data)
			return asset, nil
		}
	}

	return Asset{}, os.ErrNotExist
}

func (this *GoAssets) DecodeAsset(assetData string) ([]byte, error) {
	return decodeAsset(assetData)
}

func (this *GoAssets) AddAsset(path string, modTime time.Time, data string) {
	asset := Asset{
		Path:    path,
		ModTime: modTime,
		Data:    data,
	}

	this.assets = append(this.assets, asset)
}

func (this *GoAssets) SetAssets(assets []Asset) {
	this.assets = assets
}

func (this *GoAssets) GetAssets() []Asset {
	return this.assets
}

func (this *GoAssets) GetAssetPaths() []string {
	return this.assetPaths
}

func (this *GoAssets) SetAssetPaths(assetPaths []string) {
	this.assetPaths = assetPaths
}

func (this *GoAssets) AddAssetPath(assetPath string) {
	this.assetPaths = append(this.assetPaths, assetPath)
}

func (this *GoAssets) RemoveAssetPath(assetPath string) {
	this.assetPaths = gostring.Helper.RemoveSlice(this.assetPaths, assetPath)
}

func (this *GoAssets) SetExts(exts []string) {
	this.exts = exts
}

func (this *GoAssets) AddExt(ext string) {
	this.exts = append(this.exts, ext)
}

func (this *GoAssets) RemoveExt(ext string) {
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

	t := template.New("asset.tpl").Funcs(template.FuncMap{
		"output": func(data string) template.HTML {
			return template.HTML(data)
		},
	})
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
	var files []string
	for _, assetPath := range this.assetPaths {
		var err error
		var allowedFiles []string
		path := cleanJoinPath(this.root, assetPath)
		if len(this.exts) > 0 {
			_, allowedFiles, err = gofile.Helper.AllowedFiles(path, this.exts)
		} else {
			_, allowedFiles, err = gofile.Helper.Files(path)
		}

		if err != nil {
			return err
		}

		files = append(files, allowedFiles...)
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

	asset.Data = res

	return asset, nil
}

func cleanJoinPath(paths ...string) string {
	return cleanPath(filepath.Join(paths...))
}
func cleanPath(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
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
