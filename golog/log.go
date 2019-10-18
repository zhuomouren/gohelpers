package golog

import (
	"fmt"
	"log"
)

var (
	Level = DEBUG
	f     = logf
)

type LogLevel int

const (
	DEBUG = LogLevel(1)
	INFO  = LogLevel(2)
	WARN  = LogLevel(3)
	ERROR = LogLevel(4)
	FATAL = LogLevel(5)
)

type LogFunc func(lvl LogLevel, f string, args ...interface{})

func (l LogLevel) String() string {
	switch l {
	case 1:
		return "DEBUG"
	case 2:
		return "INFO"
	case 3:
		return "WARNING"
	case 4:
		return "ERROR"
	case 5:
		return "FATAL"
	}
	panic("invalid LogLevel")
}

func Debug(format string, args ...interface{}) {
	f(DEBUG, format, args...)
}

func Info(format string, args ...interface{}) {
	f(INFO, format, args...)
}

func Warning(format string, args ...interface{}) {
	f(WARN, format, args...)
}

func Error(format string, args ...interface{}) {
	f(ERROR, format, args...)
}

func Fatal(format string, args ...interface{}) {
	f(ERROR, format, args...)
}

func Func(ff LogFunc) {
	f = ff
}

func Logf(lvl LogLevel, format string, args ...interface{}) {
	f(lvl, format, args...)
}

// 默认日志函数
func logf(lvl LogLevel, f string, args ...interface{}) {
	if lvl < Level {
		return
	}

	log.Println(fmt.Sprintf(lvl.String()+" "+f, args...))
}
