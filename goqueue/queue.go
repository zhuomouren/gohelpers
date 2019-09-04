// goqueue 是由Go语言开发的轻量级队列系统。
package goqueue

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nsqio/go-diskqueue"
)

// 对diskqueue的日志类型进一步封装，不对外暴露diskqueue的日志类型
type LogLevel diskqueue.LogLevel

func (l LogLevel) String() string {
	return diskqueue.LogLevel(l).String()
}

const (
	DEBUG = diskqueue.DEBUG
	INFO  = diskqueue.INFO
	WARN  = diskqueue.WARN
	ERROR = diskqueue.ERROR
	FATAL = diskqueue.FATAL
)

type AppLogFunc func(lvl LogLevel, f string, args ...interface{})

const (
	MIN_MSG_SIZE int32 = 1
	MAX_MSG_SIZE int32 = 1024
)

// GoQueue 是对 DiskQueue 的封装，可以在内存和文件系统上切换
type GoQueue struct {
	sync.RWMutex
	Name            string
	MemoryQueueSize int64 // 内存里的消息数
	memoryCount     int64
	memoryMsgChan   chan []byte
	exitFlag        bool // 退出标识
	exitMutex       sync.RWMutex
	DiskQueue       bool
	DataPath        string
	MaxBytesPerFile int64         // 每个磁盘队列文件的字节数
	SyncEvery       int64         // number of writes per fsync
	SyncTimeout     time.Duration // duration of time per fsync
	backend         diskqueue.Interface
	logf            diskqueue.AppLogFunc
}

// NewGoQueue 创建一个新的GoQueue实例，并返回指针
func NewGoQueue(name string, options ...func(*GoQueue)) *GoQueue {
	gq := &GoQueue{
		Name: name,
	}
	gq.Init()

	for _, f := range options {
		f(gq)
	}

	if gq.MemoryQueueSize == 0 {
		gq.DiskQueue = true
	}

	// 使用内存存储
	if !gq.DiskQueue {
		gq.memoryMsgChan = make(chan []byte, gq.MemoryQueueSize)
		return gq
	} else {
		gq.MemoryQueueSize = 0
	}

	// 使用 diskqueue 时默认文件路径是 goqueue
	if gq.DataPath == "" {
		gq.DataPath = "goqueue"
	}

	gq.backend = diskqueue.New(
		gq.Name,
		gq.DataPath,
		gq.MaxBytesPerFile,
		MIN_MSG_SIZE,
		MAX_MSG_SIZE,
		gq.SyncEvery,
		gq.SyncTimeout,
		gq.logf,
	)

	// 创建存储路径
	if err := gq.diskQueuePath(); err != nil {
		gq.logf(ERROR, "GOQUEUE(%s): failed to create directory[%s] - %s", gq.Name, gq.DataPath, err)
	}

	return gq
}

// 队列存放路径
func DataPath(dataPath string) func(*GoQueue) {
	return func(gq *GoQueue) {
		gq.DataPath = dataPath
	}
}

// 每个磁盘队列文件的字节数
func MaxBytesPerFile(maxBytesPerFile int64) func(*GoQueue) {
	return func(gq *GoQueue) {
		gq.MaxBytesPerFile = maxBytesPerFile
	}
}

// 内存里的消息数
func MemoryQueueSize(memoryQueueSize int64) func(*GoQueue) {
	return func(gq *GoQueue) {
		gq.MemoryQueueSize = memoryQueueSize
	}
}

// 使用文件系统
func DiskQueue() func(*GoQueue) {
	return func(gq *GoQueue) {
		gq.DiskQueue = true
	}
}

func SyncEvery(syncEvery int64) func(*GoQueue) {
	return func(gq *GoQueue) {
		gq.SyncEvery = syncEvery
	}
}

func SyncTimeout(syncTimeout time.Duration) func(*GoQueue) {
	return func(gq *GoQueue) {
		gq.SyncTimeout = syncTimeout
	}
}

func Logf(logf AppLogFunc) func(*GoQueue) {
	return func(gq *GoQueue) {
		dqLogf := func(lvl diskqueue.LogLevel, f string, args ...interface{}) {
			logf(LogLevel(lvl), f, args...)
		}

		gq.logf = dqLogf
	}
}

func (gq *GoQueue) Init() {
	gq.MemoryQueueSize = 10
	gq.MaxBytesPerFile = 1024 * 1024
	gq.DiskQueue = false
	gq.SyncEvery = 1 // 2500
	gq.SyncTimeout = 2 * time.Second
	gq.logf = logf
}

// 返回一个bool，指示此队列是否关闭/退出
func (gq *GoQueue) Exiting() bool {
	return gq.exitFlag
}

// 删除一个队列，如果使用 DiskQueue 持久化，则一并删除
func (gq *GoQueue) Delete() error {
	return gq.exit(true)
}

// 关闭一个队列，不会删除 DiskQueue 持久化的数据
func (gq *GoQueue) Close() error {
	return gq.exit(false)
}

// 退出当前队列，并指示是否删除持久化的数据
func (gq *GoQueue) exit(deleted bool) error {
	gq.exitMutex.Lock()
	defer gq.exitMutex.Unlock()

	gq.exitFlag = true

	if deleted {
		gq.logf(INFO, "GOQUEUE(%s): deleting", gq.Name)
	} else {
		gq.logf(INFO, "GOQUEUE(%s): closing", gq.Name)
	}

	gq.Empty()

	if !gq.DiskQueue {
		return nil
	}

	// 删除持久化数据
	if deleted {
		return gq.backend.Delete()
	}

	return gq.backend.Close()
}

// 清空队列
func (gq *GoQueue) Empty() error {
	gq.Lock()
	defer gq.Unlock()

	gq.memoryCount = 0

	for {
		select {
		case <-gq.memoryMsgChan:
		default:
			goto finish
		}
	}

finish:
	if !gq.DiskQueue {
		return nil
	}
	return gq.backend.Empty()
}

func (gq *GoQueue) Depth() int64 {
	if gq.DiskQueue {
		return gq.backend.Depth()
	} else {
		return int64(len(gq.memoryMsgChan))
	}
}

// Depth() 的别名
func (gq *GoQueue) Size() int64 {
	return gq.Depth()
}

// 从队列中读取数据
func (gq *GoQueue) Get() ([]byte, bool) {
	var data []byte
	var ok bool

	for {
		select {
		case data, ok = <-gq.ReadChan():
			gq.logf(DEBUG, "GOQUEUE(%s): read a message - %s", gq.Name, string(data))
			goto exit
		default:
			if gq.Depth() > 0 {
				continue
			} else {
				ok = false
				goto exit
			}
		}
	}

exit:
	return data, ok
}

func (gq *GoQueue) ReadChan() chan []byte {
	if gq.DiskQueue {
		return gq.backend.ReadChan()
	} else {
		return gq.memoryMsgChan
	}
}

// 将数据写入队列
func (gq *GoQueue) Put(data []byte) error {
	gq.RLock()
	defer gq.RUnlock()
	if gq.Exiting() {
		return errors.New("exiting")
	}

	if gq.DiskQueue {
		err := gq.backend.Put(data)
		if err != nil {
			gq.logf(ERROR, "GOQUEUE(%s): failed to write message to backend - %s", gq.Name, err)
			return err
		}
	} else {
		// 如果写入队列的数量超过定义的数量，则丢弃数据，不再继续写入
		if gq.memoryCount >= gq.MemoryQueueSize {
			return nil
		}

		gq.memoryMsgChan <- data
		gq.memoryCount++
	}

	return nil
}

// 创建 DiskQueue 数据的保存路径
func (gq *GoQueue) diskQueuePath() error {
	if !gq.DiskQueue {
		return nil
	}

	path := gq.DataPath
	if !filepath.IsAbs(path) {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			return err
		}

		path = filepath.Join(dir, path)
	}

	if _, err := os.Stat(path); err == nil {
		return nil
	}

	if err := os.MkdirAll(path, 0711); err != nil {
		return err
	}

	return nil
}

// 默认日志函数，只记录 ERROR 和 FATAL 类型
func logf(lvl diskqueue.LogLevel, f string, args ...interface{}) {
	//	if lvl < diskqueue.ERROR {
	//		return
	//	}

	log.Println(fmt.Sprintf(lvl.String()+": "+f, args...))
}
