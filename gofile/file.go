package gofile

import (
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/zhuomouren/gohelpers/gostring"
)

type GoFile struct{}

var Helper = &GoFile{}

func (this GoFile) CleanJoinPath(paths ...string) string {
	return this.CleanPath(filepath.Join(paths...))
}
func (this GoFile) CleanPath(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}

// 判断文件或目录是否存在
func (this GoFile) IsExist(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}

	return false
}

// 判断文件或目录是否存在
func (this GoFile) Exists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}

	return false
}

// 创建文件夹
func (this GoFile) Mkdir(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

// 删除文件夹
func (this GoFile) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// 读取文件内容
func (this GoFile) Get(filename string) (data []byte, e error) {
	return ioutil.ReadFile(filename)
}

// 将内容写入文件，如果文件不存在，将会创建文件
func (this GoFile) Put(filename string, content []byte) error {
	if err := this.Mkdir(filepath.Dir(filename)); err != nil {
		return err
	}

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

func (this GoFile) Basename(path string) string {
	ext := filepath.Ext(path)
	filename := filepath.Base(path)

	return strings.TrimRight(filename, ext)
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

// 排除指定路径下的特定类型的文件夹和文件
func (this GoFile) BlockedFiles(root string, blockedExt []string) (dirs, files []string, err error) {
	err = filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			dirs = append(dirs, path)
		} else {
			name := fi.Name()
			if !fi.IsDir() && !strings.HasPrefix(name, ".") && !gostring.Helper.InSlice(filepath.Ext(name), blockedExt) {
				files = append(files, path)
			}
		}

		return nil
	})

	return dirs, files, err
}

func (this GoFile) CRC32(f *os.File) (uint32, error) {
	h := crc32.NewIEEE()
	_, err := io.Copy(h, f)
	f.Seek(0, 0)

	return h.Sum32(), err
}
