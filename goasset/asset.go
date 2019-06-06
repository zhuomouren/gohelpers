// 变量程序生成，增加 Get 方法获取数据
// 生成的文件添加解码函数
// 打包资源文件
package goasset

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"html/template"
	"io/ioutil"
	"os"
	"strings"

	"github.com/zhuomouren/gohelpers/gofile"
	"github.com/zhuomouren/gohelpers/gostring"
)

type GoAsset struct {
	PackageName  string
	VariableName string
	Root         string
	Exts         []string
	files        map[string]string
}

func NewGoAsset(root string) *GoAsset {
	goAsset := &GoAsset{
		Root: root,
	}

	return goAsset
}

// 写入指定的路径
func (this *GoAsset) Write(path string) error {
	return nil
}

func (this *GoAsset) SetExts(exts []string) {
	this.Exts = exts
}

func (this *GoAsset) AddExts(ext string) {
	this.Exts = append(this.Exts, ext)
}

func (this *GoAsset) RemoveExts(ext string) {
	this.Exts = gostring.Helper.RemoveSlice(this.Exts, ext)
}

func (this *GoAsset) Build() error {
	var err error

	t := template.New("asset.tpl")
	t, err = t.Parse(assetTmpl)
	if err != nil {
		return err
	}

	packageName := "main"
	if len(this.PackageName) > 0 {
		packageName = this.PackageName
	}

	items, err := this.getMaps()
	if err != nil {
		return err
	}

	w := &bytes.Buffer{}
	t.Execute(w, map[string]interface{}{
		"PackageName": packageName,
		"Items":       items,
	})

	if err := ioutil.WriteFile("asset.go", w.Bytes(), os.ModePerm); err != nil {
		return err
	}

	return nil
}

func (this *GoAsset) getMaps() (map[string]template.HTML, error) {
	items := make(map[string]template.HTML, 0)
	var files []string
	var err error

	if len(this.Exts) > 0 {
		_, files, err = gofile.FileHelper.AllowedFiles(this.Root, this.Exts)
	} else {
		_, files, err = gofile.FileHelper.Files(this.Root)
	}

	if err != nil {
		return items, err
	}

	for _, file := range files {
		data, err := this.readFile(file)
		if err != nil {
			return items, err
		}

		key := strings.Replace(file, this.Root, "", -1)
		key = strings.TrimLeft(key, string(os.PathSeparator))
		items[key] = template.HTML(data)
	}

	return items, nil
}

func (this *GoAsset) readFile(assetFile string) (string, error) {
	f, err := os.Open(assetFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// fi, err := f.Stat()
	// if err != nil {
	// 	return "", err
	// }

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	gw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return "", err
	}
	if _, err := gw.Write(data); err != nil {
		return "", err
	}
	if err := gw.Close(); err != nil {
		return "", err
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

	return res, nil
}
