// func main() {
// 	rand.Seed(time.Now().Unix())

// 	var data []interface{}
// 	for i := 0; i < 50; i++ {
// 		data = append(data, "http://www.buxutoukan.com")
// 	}

// 	fmt.Println("start ...")
// 	t1 := time.Now()
// 	gopipe.New(data, get, 1).Start().Wait()
// 	elapsed := time.Since(t1)
// 	fmt.Println("elapsed: ", elapsed)
// 	fmt.Println("ok.")
// }

// func get(val interface{}) error {
// 	url := val.(string)
// 	return gohelpers.HTTP.GET(url).Error()
// }

package gopipe

import (
	"sync"
)

type Status int

const (
	StatusOriginal Status = 0
	StatusStarted  Status = 1
	StatusStopped  Status = 2
)

type Worker func(arg interface{}) error

type Pipe struct {
	tasks       []interface{}
	worker      Worker
	concurrency uint // 并发量
	ticket      Ticket
	stopSign    bool
	status      Status
	exitFlag    chan bool
	wg          sync.WaitGroup
	err         error
}

func New(tasks []interface{}, worker Worker, concurrency uint) *Pipe {
	if concurrency <= 0 {
		concurrency = 1
	}

	if concurrency > uint(len(tasks)) {
		concurrency = uint(len(tasks))
	}

	this := &Pipe{
		tasks:       tasks,
		worker:      worker,
		concurrency: concurrency,
		stopSign:    false,
		exitFlag:    make(chan bool),
		status:      StatusOriginal,
	}

	this.ticket = NewTicket(this.concurrency)
	return this
}

func (this *Pipe) call(param interface{}) {
	defer func() {
		this.ticket.Return()
		this.wg.Done()
	}()

	// 快速执行剩余项目
	if this.stopSign {
		return
	}

	if err := this.worker(param); err != nil {
		this.err = err
		this.Stop()
	}
}

func (this *Pipe) start() {
	for _, task := range this.tasks {
		this.ticket.Take()
		this.wg.Add(1)
		go this.call(task)
	}

	this.wg.Wait()
	this.exitFlag <- true
}

func (this *Pipe) Start() *Pipe {
	this.status = StatusStarted

	go func() {
		this.start()
		this.status = StatusStopped
	}()

	return this
}

func (this *Pipe) Stop() *Pipe {
	this.status = StatusStopped
	this.stopSign = true
	return this
}

func (this *Pipe) Status() Status {
	return this.status
}

func (this *Pipe) Wait() *Pipe {
	if this.status == StatusOriginal {
		return this
	}

	<-this.exitFlag
	return this
}

func (this *Pipe) Error() error {
	return this.err
}
