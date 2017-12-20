package sync

import (
	"sync/atomic"
)

func NewRingBuffer(size int) *RingBuffer {
	p := &RingBuffer{}
	p.Init(size)
	return p
}

//goroutine-safe cycle buffer
//http://mechanitis.blogspot.com/2011/06/dissecting-disruptor-whats-so-special.html
type RingBuffer struct {
	size     int      //buffer size, readonly
	wlReader WaitList //waitlist that are wating read
	wlWriter WaitList //waitlist that are wating write
	rReserve uint64   //read reserve, mutable
	rCommit  uint64   //read commit, mutable
	wReserve uint64   //Write reserve, mutable
	wCommit  uint64   //write commit, mutable
}

func (rb *RingBuffer) Init(size int) {
	if size <= 0 {
		panic("RingBuffer: invalid size")
	}
	rb.size = size
}

func (rb *RingBuffer) Size() int {
	return rb.size
}

func (rb *RingBuffer) BufferIndex(id uint64) int {
	return int(id % uint64(rb.size))
}

func (rb *RingBuffer) ReserveW() (id uint64) {
	id = atomic.AddUint64(&rb.wReserve, 1) - 1
	for {
		dataStart := atomic.LoadUint64(&rb.rCommit)
		maxW := dataStart + uint64(rb.size)
		if id < maxW { //no conflict, reserve ok
			break
		}

		//buffer full, wait as writer in order to awake by another reader
		rb.wlWriter.Wait(PriorityFirst)
	}
	return
}
func (rb *RingBuffer) CommitW(id uint64) {
	newId := id + 1

	for {
		priority := rb.priority(id)
		if atomic.CompareAndSwapUint64(&rb.wCommit, id, newId) { //commit OK
			rb.wlReader.Wakeup(priority) //wakeup reader
			break
		}

		//commit fail, wait as reader in order to wakeup by another writer
		rb.wlReader.Wait(priority)
	}
}

func (rb *RingBuffer) ReserveR() (id uint64) {
	id = atomic.AddUint64(&rb.rReserve, 1) - 1

	for {
		w := atomic.LoadUint64(&rb.wCommit)
		if id < w { //no conflict, reserve ok
			break
		}

		//buffer empty, wait as reader in order to wakeup by another writer
		rb.wlReader.Wait(PriorityFirst)
	}
	return
}

func (rb *RingBuffer) CommitR(id uint64) {
	newId := id + 1

	for {
		priority := rb.priority(id)
		if atomic.CompareAndSwapUint64(&rb.rCommit, id, newId) {
			rb.wlWriter.Wakeup(priority) //wakeup writer
			break
		}

		//commit fail, wait as writer in order to wakeup by another reader
		rb.wlWriter.Wait(priority)
	}
}

func (rb *RingBuffer) priority(id uint64) PriorityType {
	dataStart := atomic.LoadUint64(&rb.rCommit)
	priority := PriorityType(id - dataStart)
	return priority
}
