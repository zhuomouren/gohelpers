package golog

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	DevelopmentMode bool   = true
	Filename        string // 日志文件路径
	MaxSize         int    // 每个日志的大小，单位是M
	MaxAge          int    // 文件被保存的天数
	Compress        bool   // 是否压缩
	MaxBackups      int    // 保存多少个文件备份

	logger *zap.Logger
)

var (
	std = New("")
)

func SetLogger(l *zap.Logger) {
	if logger == nil {
		logger = l
	}
}

func New(service string) *Logger {
	return &Logger{
		Service: service,
	}
}

type Logger struct {
	Service string
}

// 日志文件路径
func (this *Logger) zap() *zap.Logger {
	if logger == nil {
		logger = newZap().zap()
	}
	return logger
}

func Debug(msg string, fields ...zap.Field) {
	std.Debug(msg, fields...)
}
func (this *Logger) Debug(msg string, fields ...zap.Field) {
	this.zap().Debug(msg, this.fields(fields...)...)
}

func Info(msg string, fields ...zap.Field) {
	std.Info(msg, fields...)
}
func (this *Logger) Info(msg string, fields ...zap.Field) {
	this.zap().Debug(msg, this.fields(fields...)...)
}

func Warn(msg string, fields ...zap.Field) {
	std.Warn(msg, fields...)
}
func (this *Logger) Warn(msg string, fields ...zap.Field) {
	this.zap().Debug(msg, this.fields(fields...)...)
}

func Error(msg string, fields ...zap.Field) {
	std.Error(msg, fields...)
}
func (this *Logger) Error(msg string, fields ...zap.Field) {
	this.zap().Debug(msg, this.fields(fields...)...)
}

func Panic(msg string, fields ...zap.Field) {
	std.Panic(msg, fields...)
}
func (this *Logger) Panic(msg string, fields ...zap.Field) {
	this.zap().Debug(msg, this.fields(fields...)...)
}

func Fatal(msg string, fields ...zap.Field) {
	std.Fatal(msg, fields...)
}
func (this *Logger) Fatal(msg string, fields ...zap.Field) {
	this.zap().Debug(msg, this.fields(fields...)...)
}

// 包装 zap.Field
func String(key string, val string) zap.Field {
	return zap.String(key, val)
}
func (this *Logger) String(key string, val string) zap.Field {
	return zap.String(key, val)
}

func Int(key string, val int) zap.Field {
	return zap.Int(key, val)
}
func (this *Logger) Int(key string, val int) zap.Field {
	return zap.Int(key, val)
}

func Int32(key string, val int32) zap.Field {
	return zap.Int32(key, val)
}
func (this *Logger) Int32(key string, val int32) zap.Field {
	return zap.Int32(key, val)
}

func Int64(key string, val int64) zap.Field {
	return zap.Int64(key, val)
}
func (this *Logger) Int64(key string, val int64) zap.Field {
	return zap.Int64(key, val)
}

func Uint(key string, val uint) zap.Field {
	return zap.Uint(key, val)
}
func (this *Logger) Uint(key string, val uint) zap.Field {
	return zap.Uint(key, val)
}

func Uint32(key string, val uint32) zap.Field {
	return zap.Uint32(key, val)
}
func (this *Logger) Uint32(key string, val uint32) zap.Field {
	return zap.Uint32(key, val)
}

func Uint64(key string, val uint64) zap.Field {
	return zap.Uint64(key, val)
}
func (this *Logger) Uint64(key string, val uint64) zap.Field {
	return zap.Uint64(key, val)
}

func Time(key string, val time.Time) zap.Field {
	return zap.Time(key, val)
}
func (this *Logger) Time(key string, val time.Time) zap.Field {
	return zap.Time(key, val)
}

func Duration(key string, val time.Duration) zap.Field {
	return zap.Duration(key, val)
}
func (this *Logger) Duration(key string, val time.Duration) zap.Field {
	return zap.Duration(key, val)
}

func Reflect(key string, val interface{}) zap.Field {
	return zap.Reflect(key, val)
}
func (this *Logger) Reflect(key string, val interface{}) zap.Field {
	return zap.Reflect(key, val)
}

func Any(key string, val interface{}) zap.Field {
	return zap.Any(key, val)
}
func (this *Logger) Any(key string, val interface{}) zap.Field {
	return zap.Any(key, val)
}

func (this *Logger) fields(fields ...zap.Field) []zap.Field {
	if this.Service != "" {
		fields = append(fields, zap.String("service", this.Service))
	}
	return fields
}

type zaplog struct {
	filename        string
	maxSize         int
	maxAge          int
	compress        bool
	maxBackups      int
	developmentMode bool
}

func newZap() *zaplog {
	log := &zaplog{}
	log.SetFilename(Filename)
	log.SetMaxSize(MaxSize)
	log.SetMaxAge(MaxAge)
	log.SetCompress(Compress)
	log.SetMaxBackups(MaxBackups)
	log.SetDevelopmentMode(DevelopmentMode)

	return log
}

// 日志文件路径
func (this *zaplog) SetFilename(filename string) *zaplog {
	this.filename = filename
	return this
}

// 每个日志的大小，单位是M
func (this *zaplog) SetMaxSize(maxSize int) *zaplog {
	if maxSize < 1 {
		maxSize = 1
	}
	this.maxSize = maxSize
	return this
}

// 文件被保存的天数
func (this *zaplog) SetMaxAge(maxAge int) *zaplog {
	if maxAge < 1 {
		maxAge = 1
	}
	this.maxAge = maxAge
	return this
}

// 是否压缩
func (this *zaplog) SetCompress(compress bool) *zaplog {
	this.compress = compress
	return this
}

// 保存多少个文件备份
func (this *zaplog) SetMaxBackups(maxBackups int) *zaplog {
	if maxBackups < 1 {
		maxBackups = 1
	}
	this.maxBackups = maxBackups
	return this
}

// 是否开发模式
func (this *zaplog) SetDevelopmentMode(developmentMode bool) *zaplog {
	this.developmentMode = developmentMode
	return this
}

func (this *zaplog) zap() *zap.Logger {
	hook := lumberjack.Logger{
		Filename:   this.filename,
		MaxSize:    this.maxSize,
		MaxAge:     this.maxAge,
		Compress:   this.compress,
		MaxBackups: this.maxBackups,
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	atomicLevel := zap.NewAtomicLevel()
	if this.developmentMode {
		atomicLevel.SetLevel(zap.DebugLevel)
	} else {
		atomicLevel.SetLevel(zap.InfoLevel)
	}

	var params []zapcore.WriteSyncer
	if this.developmentMode {
		params = append(params, zapcore.AddSync(os.Stdout))
		if this.filename != "" {
			params = append(params, zapcore.AddSync(&hook))
		}
	} else {
		if this.filename == "" {
			params = append(params, zapcore.AddSync(os.Stdout))
		} else {
			params = append(params, zapcore.AddSync(&hook))
		}
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(params...),
		atomicLevel,
	)

	var options []zap.Option
	options = append(options, zap.AddCaller())
	options = append(options, zap.AddCallerSkip(2))
	if this.developmentMode {
		options = append(options, zap.Development())
	}

	return zap.New(core, options...)
}
