package gofile

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/zhuomouren/gohelpers/gostring"
)

type GoFile struct{}

var FileHelper = &GoFile{}

// 判断文件或目录是否存在
func (this GoFile) IsExist(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}

	return false
}

// 创建文件夹
func (this GoFile) Mkdir(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

// 读取文件内容
func (this GoFile) Get(filename string) (data []byte, e error) {
	return ioutil.ReadFile(filename)
}

// 将内容写入文件，如果文件不存在，将会创建文件
func (this GoFile) Put(filename string, content []byte) error {
	return ioutil.WriteFile(filename, content, os.ModePerm)
}

// filenameSafe returns whether all characters in s are printable ASCII
// and safe to use in a filename on most filesystems.
func (this GoFile) FilenameSafe(s string) bool {
	for _, c := range s {
		if c < 0x20 || c > 0x7E {
			return false
		}
		switch c {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			return false
		}
	}
	return true
}

// 返回指定路径下的所有文件夹和文件
func (this GoFile) Files(root string) (dirs, files []string, err error) {
	err = filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			dirs = append(dirs, path)
		} else {
			files = append(files, path)
		}

		return nil
	})

	return dirs, files, err
}

// 返回指定路径下的特定类型的文件夹和文件
func (this GoFile) AllowedFiles(root string, allowedExt []string) (dirs, files []string, err error) {
	if len(allowedExt) == 0 {
		return dirs, files, nil
	}

	err = filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			dirs = append(dirs, path)
		} else {
			name := fi.Name()
			if !fi.IsDir() && !strings.HasPrefix(name, ".") && gostring.Helper.InSlice(filepath.Ext(name), allowedExt) {
				files = append(files, path)
			}
		}

		return nil
	})

	return dirs, files, err
}
