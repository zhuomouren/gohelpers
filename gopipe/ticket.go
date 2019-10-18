package gopipe

// Goroutine票池的接口。
type Ticket interface {
	// 拿走一张票。
	Take()
	// 归还一张票。
	Return()
	// 票池是否已被激活。
	Active() bool
	// 票的总数。
	Total() uint
	// 剩余的票数。
	Remainder() uint
}

// Goroutine票池的实现。
type ticket struct {
	total    uint      // 票的总数。
	ticketCh chan byte // 票的容器。
	active   bool      // 票池是否已被激活。
}

func NewTicket(total uint) *ticket {
	this := &ticket{}
	this.init(total)
	return this
}

func (this *ticket) init(total uint) {
	ch := make(chan byte, total)
	n := int(total)
	for i := 0; i < n; i++ {
		ch <- 1
	}
	this.ticketCh = ch
	this.total = total
	this.active = true
}

func (this *ticket) Take() {
	<-this.ticketCh
}

func (this *ticket) Return() {
	this.ticketCh <- 1
}

func (this *ticket) Active() bool {
	return this.active
}

func (this *ticket) Total() uint {
	return this.total
}

func (this *ticket) Remainder() uint {
	return uint(len(this.ticketCh))
}
