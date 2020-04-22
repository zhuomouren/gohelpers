package goini

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/zhuomouren/gohelpers/golog"
)

var stores = map[string]interface{}{}

type IniFile struct {
	sync.RWMutex
	data            []map[string]string
	filepath        string
	keys            []string
	sections        []string
	autoReload      bool
	changedCallback func()
	modTime         int64
	logger          *golog.Logger
}

func New(filepath string, options ...func(*IniFile)) *IniFile {
	ini := &IniFile{
		filepath:   filepath,
		autoReload: true,
	}
	ini.initData()

	for _, f := range options {
		f(ini)
	}

	if ini.logger == nil {
		ini.logger = golog.New("ini")
	}

	err := ini.parse()
	if err != nil {
		ini.initData()
		ini.logger.Error(fmt.Sprintf("parse ini error: %s", err.Error()))
		return ini
	}

	// 监控文件
	if ini.autoReload {
		go ini.watcher()
	}

	return ini
}

// 修改配置文件是否自动重载
func AutoReload(autoReload bool) func(*IniFile) {
	return func(ini *IniFile) {
		if fileExist(ini.filepath) {
			ini.autoReload = autoReload
		}
	}
}

// 配置文件自动重载后触发
func OnReload(f func()) func(*IniFile) {
	return func(ini *IniFile) {
		ini.changedCallback = f
	}
}

// 设置记录日志的函数
func SetLogger(log *golog.Logger) func(*IniFile) {
	return func(ini *IniFile) {
		ini.logger = log
	}
}

func (ini *IniFile) Get(key string, def ...interface{}) *Value {
	key = strings.ToLower(key)
	for _, item := range ini.data {
		if val, ok := item[key]; ok {
			return NewValue(val)
		}
	}

	ini.logger.Debug("key does not exist",
		ini.logger.String("key", key))

	return GetDefault(def...)
}

// 判断 key 是否存在
func (ini *IniFile) IsExist(key string) bool {
	key = strings.ToLower(key)
	for _, item := range ini.data {
		if _, ok := item[key]; ok {
			return true
		}
	}

	return false
}

// 返回配置文件的所有节点
func (ini *IniFile) Sections() []string {
	return ini.sections
}

// 返回配置文件的所有key
func (ini *IniFile) Keys() []string {
	return ini.keys
}

// 监控配置文件
func (ini *IniFile) watcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		ini.logger.Error("fsnotify error",
			ini.logger.String("error", err.Error()),
		)
		return
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					mt, err := getFileModTime(event.Name)
					ini.logger.Error("fsnotify error",
						ini.logger.String("error", err.Error()),
					)
					if ini.modTime == mt {
						continue
					}

					ini.modTime = mt
					if err := ini.parse(); err != nil {
						ini.logger.Error("ini parse error",
							ini.logger.String("error", err.Error()),
						)
					} else if ini.changedCallback != nil {
						ini.logger.Info("ini file has been modified")
						ini.changedCallback()
					}
				}
			case err := <-watcher.Errors:
				if err != nil {
					ini.logger.Error("fsnotify error",
						ini.logger.String("error", err.Error()),
					)
				}
			}
		}
	}()

	err = watcher.Add(ini.filepath)
	if err != nil {
		ini.logger.Error("fsnotify error",
			ini.logger.String("error", err.Error()),
		)
	}
	<-done
}

// 解析配置文件
// ; 和 # 起始的行是注释
func (ini *IniFile) parse() error {
	ini.Lock()
	defer ini.Unlock()

	if !fileExist(ini.filepath) {
		ini.logger.Warn("ini file does not exist.",
			ini.logger.String("file", ini.filepath),
		)
		ini.autoReload = false
		return nil
	}

	f, err := os.Open(ini.filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	ini.logger.Info("parse ini file")

	data := []map[string]string{}
	keys := []string{}
	sections := []string{}

	var section string
	buf := bufio.NewReader(f)
	lineNum := 0
	for done := false; !done; {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				done = true
			} else {
				return err
			}
		}

		lineNum++

		line = strings.TrimSpace(line)
		// 空行或注释
		if line == "" || line[0] == ';' || line[0] == '#' {
			continue
		}

		// 节点
		if line[0] == '[' && line[len(line)-1] == ']' {
			section = strings.TrimSpace(line[1 : len(line)-1])
			if section != "" {
				sections = append(sections, section)
				continue
			}
		}

		pair := strings.SplitN(line, "=", 2)
		if len(pair) != 2 {
			return newSyntaxError(ini.filepath, lineNum, line)
		}

		key := strings.ToLower(strings.TrimSpace(pair[0]))
		if section != "" {
			key = section + "." + key
		}

		// 移除头尾双引号
		val := strings.Trim(strings.TrimSpace(pair[1]), "\"")
		if key == "" || val == "" {
			return newSyntaxError(ini.filepath, lineNum, line)
		}

		// 处理双引号
		if val[0] == '"' {
			val = strings.Trim(val, "\"")
			val = strings.Replace(val, `""`, `"`, -1)
			val = strings.Replace(val, `\"`, `"`, -1)
		}

		ini.logger.Debug("parse file info", ini.logger.String(key, val))

		item := map[string]string{
			key: val,
		}

		keys = append(keys, key)
		data = append(data, item)
	}

	ini.data, ini.sections, ini.keys = data, sections, keys

	ini.logger.Debug("ini file info",
		ini.logger.Int("count", len(ini.data)),
		ini.logger.Int("sections", len(ini.sections)),
		ini.logger.Int("keys", len(ini.keys)),
	)

	return nil
}

// 重置内部数据
func (ini *IniFile) initData() {
	ini.data = []map[string]string{}
	ini.keys = []string{}
	ini.sections = []string{}
}

// 判断文件是否存在
func fileExist(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}

	return false
}

// 返回文件最后修改时间戳
func getFileModTime(path string) (int64, error) {
	path = filepath.ToSlash(filepath.Clean(path))
	f, err := os.Open(path)
	if err != nil {
		return time.Now().Unix(), fmt.Errorf("Fail to open file[ %s ]", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return time.Now().Unix(), fmt.Errorf("Fail to get file information[ %s ]", err)
	}

	return fi.ModTime().Unix(), nil
}
