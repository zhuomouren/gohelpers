package goini

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	once sync.Once
	std  *IniFile

	RootPath   string
	ConfigFile string = "app.conf"
)

func new() *IniFile {
	once.Do(func() {
		var err error

		if len(RootPath) == 0 {
			if RootPath, err = filepath.Abs(filepath.Dir(os.Args[0])); err != nil {
				panic(err)
			}
		}

		path := filepath.Join(RootPath, "conf", ConfigFile)
		if !fileExist(path) {
			path = filepath.Join(RootPath, ConfigFile)
		}
		std = New(path)
	})

	return std
}

func Get(key string, def ...interface{}) *Value {
	return new().Get(key, def...)
}

func IsExist(key string) bool {
	return new().IsExist(key)
}

func Sections() []string {
	return new().Sections()
}

func Keys() []string {
	return new().Keys()
}
