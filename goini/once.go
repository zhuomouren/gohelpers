package goini

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	once sync.Once
	cfg  *IniFile

	RootPath   string
	ConfigFile string = "app.conf"
)

func init() {
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
		cfg = New(path, SetLogLevel(ERROR))
	})
}

func Get(key string, def ...interface{}) *Value {
	return cfg.Get(key, def...)
}

func IsExist(key string) bool {
	return cfg.IsExist(key)
}

func Sections() []string {
	return cfg.Sections()
}

func Keys() []string {
	return cfg.Keys()
}
