//// author Ally Dale<vipally@gmail.com>
// path : goroot/runtime/sema_export_ally.go

package runtime

import (
	"unsafe"
)

//type of goroutin priority
type priorityType = uint64

const (
	priorityFirst priorityType = 0
	priorityLast  priorityType = 1<<64 - 1
)

//// Semacquire waits until *s > 0 and then atomically decrements it.
//// It is intended as a simple sleep primitive for use by the synchronization
//// library and should not be used directly.
//func Semacquire(addr *uint32) {
//	sync_runtime_Semacquire(addr)
//}

//// SemacquireMutex is like Semacquire, but for profiling contended Mutexes.
//// If lifo is true, queue waiter at the head of wait queue.
//func SemacquireMutex(addr *uint32, lifo bool) {
//	sync_runtime_SemacquireMutex(addr, lifo)
//}

//// Semrelease atomically increments *s and notifies a waiting goroutine
//// if one is blocked in Semacquire.
//// It is intended as a simple wakeup primitive for use by the synchronization
//// library and should not be used directly.
//// If handoff is true, pass count directly to the first waiter.
//func Semrelease(addr *uint32, handoff bool) {
//	sync_runtime_Semrelease(addr, handoff)
//}

//// add g to waitSem list sort by priority ascend, and wake up head of awakeSem
//// if waitSem is nil, g will be added to global sched list
//// if awakeSem is nil, it will call Gosched
//func GoschedAndAwake(waitSem *uint32, priority uint32, awakeSem *uint32) {
//	goschedBlockWithPriority(waitSem, priority)
//	goschedWakeUp(awakeSem)
//	if false {
//		goschedImpl(getg())
//		Gosched()
//	}
//}

// add g to waitSem list sort by priority ascend.
// if waitSem is nil, g will be added to global sched list
//go:linkname sync_goWaitWithPriority sync.goWaitWithPriority
func sync_goWaitWithPriority(waitSem *uint32, priority priorityType) {
	if waitSem != nil {
		s := acquireSudog()
		root := semroot(waitSem)
		lock(&root.lock)
		root.queueWithPriority(waitSem, s, priority)
		unlock(&root.lock)
		//goparkunlock(&root.lock, "semacquire", traceEvGoBlockSync, 4)
	} else {
		panic("nil waitSem")
		//gp := getg()
		//		status := readgstatus(gp)
		//		if status&^_Gscan != _Grunning {
		//			dumpgstatus(gp)
		//			throw("bad g status")
		//		}
		//		casgstatus(gp, _Grunning, _Grunnable)
		//		dropg()
		//		lock(&sched.lock)
		//		globrunqput(gp)
		//		unlock(&sched.lock)
	}
	schedule()
}

// wake up gs which hold pri <= priority from head of awakeSem.
// Current g will continue.
//go:linkname sync_goAwakeWithPriority sync.goAwakeWithPriority
func sync_goAwakeWithPriority(awakeSem *uint32, priority priorityType) {
	if awakeSem != nil {
		root := semroot(awakeSem)
		lock(&root.lock)
		s, _ := root.dequeue(awakeSem)
		unlock(&root.lock)
		if s != nil {
			readyWithTime(s, 5)
		} else {
			schedule()
		}
	} else {
		panic("nil awakeSem")
		//schedule()
	}
}

func (root *semaRoot) dequeueWithPriority(addr *uint32, priority priorityType) (n int) {
	return 0
}

// queue adds s to the blocked goroutines in semaRoot with priority.
func (root *semaRoot) queueWithPriority(addr *uint32, s *sudog, priority priorityType) {
	if false { //debug refer
		root.queue(addr, s, false)
		root.dequeue(addr)
	}

	s.g = getg()
	s.elem = unsafe.Pointer(addr)
	s.next = nil
	s.prev = nil
	s.priority = priority //set the wait link priority

	var last *sudog
	pt := &root.treap
	for t := *pt; t != nil; t = *pt {
		if t.elem == unsafe.Pointer(addr) {
			// Already have addr in list.
			if priority == priorityFirst || priority < t.priority {
				// Substitute s in t's place in treap.
				*pt = s
				s.ticket = t.ticket
				s.acquiretime = t.acquiretime
				s.parent = t.parent
				s.prev = t.prev
				s.next = t.next
				if s.prev != nil {
					s.prev.parent = s
				}
				if s.next != nil {
					s.next.parent = s
				}
				// Add t first in s's wait list.
				s.waitlink = t
				s.waittail = t.waittail
				if s.waittail == nil {
					s.waittail = t
				}
				t.parent = nil
				t.prev = nil
				t.next = nil
				t.waittail = nil
			} else if priority == priorityLast || t.waitlink == nil {
				// Add s to end of t's wait list.
				if t.waittail == nil {
					t.waitlink = s
				} else {
					t.waittail.waitlink = s
				}
				t.waittail = s
				s.waitlink = nil
			} else { // add s to wait list order by priority ascend
				p := *pt
				for ; p.waitlink != nil && priority < p.waitlink.priority; p = p.waitlink { //find the suitable node to insert after
					//do nothing
				}
				n := p.waitlink
				p.waitlink = s
				s.parent = nil
				s.waittail = nil
				if s.waitlink = n; n == nil {
					t.waittail = s
				}
			}
			return
		}
		last = t
		if uintptr(unsafe.Pointer(addr)) < uintptr(t.elem) {
			pt = &t.prev
		} else {
			pt = &t.next
		}
	}

	// Add s as new leaf in tree of unique addrs.
	// The balanced tree is a treap using ticket as the random heap priority.
	// That is, it is a binary tree ordered according to the elem addresses,
	// but then among the space of possible binary trees respecting those
	// addresses, it is kept balanced on average by maintaining a heap ordering
	// on the ticket: s.ticket <= both s.prev.ticket and s.next.ticket.
	// https://en.wikipedia.org/wiki/Treap
	// http://faculty.washington.edu/aragon/pubs/rst89.pdf
	s.ticket = fastrand()
	s.parent = last
	*pt = s

	// Rotate up into tree according to ticket (priority).
	for s.parent != nil && s.parent.ticket > s.ticket {
		if s.parent.prev == s {
			root.rotateRight(s.parent)
		} else {
			if s.parent.next != s {
				panic("semaRoot queue")
			}
			root.rotateLeft(s.parent)
		}
	}
}

func GetGoroutineId() int64 {
	return getg().goid
}

//func GoschedWait(priority uint32) {
//	_g_ := getg()
//	_g_.priority = priority
//	mcall(gosched_wait)
//}
//func goschedWaitRelease() {
//	lock(&sched.lock)
//	globwaitqrelease()
//	unlock(&sched.lock)
//}

//func gosched_wait(gp *g) {
//	if trace.enabled {
//		traceGoSched()
//	}
//	goschedWaitImpl(gp)
//}

//func goschedWaitImpl(gp *g) {
//	println("goschedWaitImpl", gp.goid, "wait size=", sched.waitqsize)
//	status := readgstatus(gp)
//	if status&^_Gscan != _Grunning {
//		dumpgstatus(gp)
//		throw("bad g status")
//	}
//	casgstatus(gp, _Grunning, _Gwaiting)
//	dropg()
//	lock(&sched.lock)
//	globwaitqput(gp)
//	unlock(&sched.lock)

//	releasem(acquirem()) //added

//	//schedule(true)
//}

//func globwaitqput(gp *g) {
//	gp.schedlink = 0
//	if sched.waitqtail != 0 {
//		sched.waitqtail.ptr().schedlink.set(gp)
//	} else {
//		sched.waitqhead.set(gp)
//	}
//	sched.waitqtail.set(gp)
//	sched.waitqsize++
//}

//func globwaitqrelease() {
//	println("globwaitqrelease ", "wait size=", sched.waitqsize)
//	h := sched.waitqhead.ptr()
//	t := sched.waitqtail.ptr()
//	size := sched.waitqsize
//	if h == nil {
//		return
//	}
//	for p := sched.waitqhead.ptr(); p != nil; p = p.schedlink.ptr() {
//		casgstatus(p, _Gwaiting, _Grunnable)
//	}
//	if sched.runqtail != 0 {
//		sched.runqtail.ptr().schedlink.set(h)
//	} else {
//		sched.runqhead.set(h)
//	}
//	sched.runqtail.set(t)
//	sched.runqsize += size

//	sched.waitqhead = 0
//	sched.waitqtail = 0
//	sched.waitqsize = 0
//}
