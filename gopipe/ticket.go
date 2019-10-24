package gopipe

type Ticket interface {
	Take()
	Return()
	Active() bool
	Total() uint
	Remainder() uint
}

type ticket struct {
	total    uint
	ticketCh chan byte
	active   bool
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
