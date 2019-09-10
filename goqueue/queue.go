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

type LogFunc func(lvl LogLevel, f string, args ...interface{})

const (
	MIN_MSG_SIZE int32 = 1   // 消息的最小长度
	MAX_MSG_SIZE int32 = 255 // 消息的最大长度
)

// goQueue 是对 DiskQueue 的封装，可以在内存和文件系统上切换
type goQueue struct {
	sync.RWMutex
	name            string
	memoryMaxSize   uint64 // 内存里允许的最大消息数量
	memoryCount     uint64 // // 当前内存里的消息数量
	memoryMsgChan   chan []byte
	exitFlag        bool // 退出标识
	exitMutex       sync.RWMutex
	useDisk         bool
	dataPath        string
	maxBytesPerFile int64         // 每个磁盘队列文件的字节数
	syncEvery       int64         // number of writes per fsync
	syncTimeout     time.Duration // duration of time per fsync
	backend         diskqueue.Interface
	logf            diskqueue.AppLogFunc
}

// 创建一个新的内存队列实例，并返回指针
func NewMemory(name string, options ...uint64) *goQueue {
	this := &goQueue{
		name:          name,
		useDisk:       false,
		memoryMaxSize: 100000, // 默认 10w
		logf:          logf,
	}

	if len(options) > 0 {
		this.memoryMaxSize = options[0]
	}

	this.memoryMsgChan = make(chan []byte, this.memoryMaxSize)
	return this
}

// 创建一个新的goQueue实例，并返回指针
func New(name string, options ...string) *goQueue {
	this := &goQueue{
		name:            name,
		useDisk:         true,
		dataPath:        "diskqueue",
		memoryMaxSize:   0,
		maxBytesPerFile: 1024 * 1024 * 10, // 默认 10M
		syncEvery:       1024,             // 2500
		syncTimeout:     2 * time.Second,
		logf:            logf,
	}

	if len(options) > 0 {
		this.dataPath = options[0]
	}

	this.backend = diskqueue.New(
		this.dataPath,
		this.dataPath,
		this.maxBytesPerFile,
		MIN_MSG_SIZE,
		MAX_MSG_SIZE,
		this.syncEvery,
		this.syncTimeout,
		this.logf,
	)

	// 创建存储路径
	if err := this.diskQueuePath(); err != nil {
		this.logf(ERROR, "GOQUEUE(%s): failed to create directory[%s] - %s",
			this.name, this.dataPath, err.Error())
	}

	return this
}

// 队列存放路径
func (this *goQueue) DataPath(dataPath string) *goQueue {
	this.dataPath = dataPath
	return this
}

// 每个磁盘队列文件的字节数
func (this *goQueue) MaxBytesPerFile(maxBytesPerFile int64) *goQueue {
	this.maxBytesPerFile = maxBytesPerFile
	return this
}

// 内存里的消息数
func (this *goQueue) MemoryMaxSize(memoryMaxSize uint64) *goQueue {
	this.memoryMaxSize = memoryMaxSize
	return this
}

func (this *goQueue) SyncEvery(syncEvery int64) *goQueue {
	this.syncEvery = syncEvery
	return this
}

func (this *goQueue) SyncTimeout(syncTimeout time.Duration) *goQueue {
	this.syncTimeout = syncTimeout
	return this
}

func (this *goQueue) Logf(logf LogFunc) *goQueue {
	dqLogf := func(lvl diskqueue.LogLevel, f string, args ...interface{}) {
		logf(LogLevel(lvl), f, args...)
	}

	this.logf = dqLogf
	return this
}

// 返回一个bool，指示此队列是否关闭/退出
func (this *goQueue) Exiting() bool {
	return this.exitFlag
}

// 删除一个队列，如果使用 DiskQueue 持久化，则一并删除
func (this *goQueue) Delete() error {
	return this.exit(true)
}

// 关闭一个队列，不会删除 DiskQueue 持久化的数据
func (this *goQueue) Close() error {
	return this.exit(false)
}

// 退出当前队列，并指示是否删除持久化的数据
func (this *goQueue) exit(deleted bool) error {
	this.exitMutex.Lock()
	defer this.exitMutex.Unlock()

	this.exitFlag = true

	if deleted {
		this.logf(INFO, "GOQUEUE(%s): deleting", this.name)
	} else {
		this.logf(INFO, "GOQUEUE(%s): closing", this.name)
	}

	this.Empty()

	if !this.useDisk {
		return nil
	}

	// 删除持久化数据
	if deleted {
		return this.backend.Delete()
	}

	return this.backend.Close()
}

// 清空队列
func (this *goQueue) Empty() error {
	this.Lock()
	defer this.Unlock()

	this.memoryCount = 0

	for {
		select {
		case <-this.memoryMsgChan:
		default:
			goto finish
		}
	}

finish:
	if !this.useDisk {
		return nil
	}
	return this.backend.Empty()
}

func (this *goQueue) Depth() int64 {
	if this.useDisk {
		return this.backend.Depth()
	} else {
		return int64(len(this.memoryMsgChan))
	}
}

// Depth() 的别名
func (this *goQueue) Size() int64 {
	return this.Depth()
}

// 从队列中读取数据
func (this *goQueue) Get() ([]byte, bool) {
	var data []byte
	var ok bool

	for {
		select {
		case data, ok = <-this.ReadChan():
			this.logf(DEBUG, "GOQUEUE(%s): read a message - %s", this.name, string(data))
			goto exit
		default:
			if this.Depth() > 0 {
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

func (this *goQueue) ReadChan() chan []byte {
	if this.useDisk {
		return this.backend.ReadChan()
	} else {
		return this.memoryMsgChan
	}
}

// 将数据写入队列
func (this *goQueue) Put(data []byte) error {
	this.RLock()
	defer this.RUnlock()
	if this.Exiting() {
		return errors.New("exiting")
	}

	if this.useDisk {
		err := this.backend.Put(data)
		if err != nil {
			this.logf(ERROR, "GOQUEUE(%s): failed to write message to backend - %s", this.name, err)
			return err
		}
	} else {
		// 如果写入队列的数量超过定义的数量，则丢弃数据，不再继续写入
		if this.memoryCount >= this.memoryMaxSize {
			return nil
		}

		this.memoryMsgChan <- data
		this.memoryCount++
	}

	return nil
}

// 创建 DiskQueue 数据的保存路径
func (this *goQueue) diskQueuePath() error {
	if !this.useDisk {
		return nil
	}

	path := this.dataPath
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
