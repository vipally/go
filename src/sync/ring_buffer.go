package sync

import (
	"sync/atomic"
)

func NewRingBuffer(size int) *RingBuffer {
	p := &RingBuffer{}
	p.Init(size)
	return p
}

// RingBuffer is goroutine-safe cycle buffer.
// It is designed as share buffer with lots of readers and writers.
// see: http://mechanitis.blogspot.com/2011/06/dissecting-disruptor-whats-so-special.html
type RingBuffer struct {
	size int //buffer size, readonly

	wlReader WaitList //waitlist that are wating read
	wlWriter WaitList //waitlist that are wating write
	rReserve uint64   //read reserve, mutable
	rCommit  uint64   //read commit, mutable
	wReserve uint64   //Write reserve, mutable
	wCommit  uint64   //write commit, mutable
}

// Init ringbuffer with size
func (rb *RingBuffer) Init(size int) {
	if size <= 0 {
		panic("RingBuffer: invalid size")
	}
	rb.size = size
}

// Size return size of ringbuffer
func (rb *RingBuffer) Size() int {
	return rb.size
}

// BufferIndex returns logic index of buffer by id
func (rb *RingBuffer) BufferIndex(id uint64) int {
	return int(id % uint64(rb.size))
}

// ReserveW returns next avable id for write.
// It will wait if ringbuffer is full.
func (rb *RingBuffer) ReserveW() (id uint64) {
	id = atomic.AddUint64(&rb.wReserve, 1) - 1
	for {
		dataStart := atomic.LoadUint64(&rb.rCommit)
		maxW := dataStart + uint64(rb.size)
		if id < maxW { //no conflict, reserve ok
			break
		}

		//buffer full, wait as writer in order to awake by another reader
		rb.wlWriter.Wait(PriorityFirst, &rb.rCommit, dataStart+1)
	}
	return
}

// CommitW commit writer event for id.
// It will wait if previous writer id havn't commit.
// It will awake on reader wait list after commit OK.
func (rb *RingBuffer) CommitW(id uint64) {
	newId := id + 1
	priority := rb.priority(id)
	for {
		if atomic.CompareAndSwapUint64(&rb.wCommit, id, newId) { //commit OK
			rb.wlReader.Wakeup(priority) //wakeup reader
			break
		}

		//commit fail, wait as reader in order to wakeup by another writer
		rb.wlReader.Wait(priority, &rb.wCommit, id)
	}
}

// ReserveR returns next avable id for read.
// It will wait if ringbuffer is empty.
func (rb *RingBuffer) ReserveR() (id uint64) {
	id = atomic.AddUint64(&rb.rReserve, 1) - 1
	for {
		w := atomic.LoadUint64(&rb.wCommit)
		if id < w { //no conflict, reserve ok
			break
		}

		//buffer empty, wait as reader in order to wakeup by another writer
		rb.wlReader.Wait(PriorityFirst, &rb.wCommit, id-1)
	}
	return
}

// CommitR commit reader event for id.
// It will wait if previous reader id havn't commit.
// It will awake on writer wait list after commit OK.
func (rb *RingBuffer) CommitR(id uint64) {
	newId := id + 1
	priority := rb.priority(id)
	for {
		if atomic.CompareAndSwapUint64(&rb.rCommit, id, newId) {
			rb.wlWriter.Wakeup(priority) //wakeup writer
			break
		}

		//commit fail, wait as writer in order to wakeup by another reader
		rb.wlWriter.Wait(priority, &rb.rCommit, id)
	}
}

func (rb *RingBuffer) priority(id uint64) PriorityType {
	return id
}
