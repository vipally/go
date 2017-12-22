//// author Ally Dale<vipally@gmail.com>
// path : goroot/runtime/sema_export_ally.go

package runtime

import (
	"runtime/internal/atomic"
	"unsafe"
)

//type of goroutin priority
type priorityType = uint64

const (
	priorityFirst priorityType = 0
	priorityLast  priorityType = 1<<64 - 1
)

// add g to waitSem list sort by priority ascend.
// if waitSem is nil, g will be added to global sched list
//go:linkname sync_runtime_goWaitWithPriority sync.runtime_goWaitWithPriority
func sync_runtime_goWaitWithPriority(waitSem *uint32, priority priorityType, needblock func() bool) {
	if waitSem == nil {
		panic("nil waitSem")
	}

	gp := getg()
	if gp != gp.m.curg {
		throw("goWaitWithPriority not on the G stack")
	}

	root := semroot(waitSem)
	lock(&root.lock)

	if needblock != nil && !needblock() { //block condition miss, do not block
		unlock(&root.lock)
		return
	}

	s := acquireSudog()
	s.releasetime = 0
	s.acquiretime = 0
	s.ticket = 0
	// Add ourselves to nwait to disable "easy case" in semrelease.
	root.debugShowList(waitSem, "wait before queue")
	atomic.Xadd(&root.nwait, 1)
	//println("before Xadd", waitSem, *waitSem, 1)
	atomic.Xadd(waitSem, 1)
	//println("after Xadd", waitSem, *waitSem, 1)
	root.queuePriority(waitSem, s, priority)
	root.debugShowList(waitSem, "wait after queue")

	unlock(&root.lock)
	gopark(nil, nil, "syncWaitList", traceEvGoBlockWaitList, 4)
	//goparkunlockWaitList(&root.lock, "syncWaitList", traceEvGoBlockWaitList, 4)

	releaseSudog(s)
}

// wake up gs which hold pri <= priority from head of awakeSem.
// Current g will continue.
//go:linkname sync_runtime_goAwakeWithPriority sync.runtime_goAwakeWithPriority
func sync_runtime_goAwakeWithPriority(awakeSem *uint32, priority priorityType) {
	if awakeSem == nil {
		panic("nil awakeSem")
	}

	gp := getg()
	if gp != gp.m.curg {
		throw("goAwakeWithPriority not on the G stack")
	}

	root := semroot(awakeSem)

	// Easy case: no waiters?
	// This check must happen after the xadd, to avoid a missed wakeup
	// (see loop in semacquire).
	if atomic.Load(&root.nwait) == 0 {
		return
	}

	// Harder case: search for a waiter and wake it.
	lock(&root.lock)
	root.debugShowList(awakeSem, "awake before dequeue")
	num := root.dequeuePriority(awakeSem, priority)
	if num > 0 {
		atomic.Xadd(&root.nwait, -num)
		//println("before Xadd dec", awakeSem, *awakeSem, -num)
		atomic.Xadd(awakeSem, -num)
		//println("after Xadd dec", awakeSem, *awakeSem, -num)
	}
	root.debugShowList(awakeSem, "awake after dequeue")

	//println("=========wakeup", awakeSem, priority, num)
	unlock(&root.lock)
}

func (root *semaRoot) debugShowList(sem *uint32, title string) {
	//	n := atomic.Load(sem)
	//	println("debugShowList", n, sem, title, 0)
	//	ps := &root.treap
	//	s := *ps
	//	for elem := uintptr(unsafe.Pointer(sem)); s != nil; s = *ps {
	//		if s.elem == unsafe.Pointer(sem) {
	//			goto Found
	//		}
	//		if elem < uintptr(s.elem) {
	//			ps = &s.prev
	//		} else {
	//			ps = &s.next
	//		}
	//	}

	//	if n > 0 && title != "semroot" {
	//		println("============waitlist len not match", sem, n, 0)
	//		root.debugShowAddrs(root.treap, sem, title, 0)
	//		throw("list len not match")
	//	}
	//	return //do not found

	//Found:
	//	cnt := uint32(0)
	//	for p := s; p != nil; p = p.waitlink {
	//		println("    ", p.g.goid, p.priority)
	//		cnt++
	//	}
	//	if cnt != n && title != "semroot" {
	//		println("============waitlist len not match", sem, n, cnt)
	//		root.debugShowAddrs(root.treap, sem, title, 0)
	//		throw("list len not match")
	//	}
}

func (root *semaRoot) debugShowAddrs(s *sudog, sem *uint32, title string, depth int) {
	//	if depth == 0 {
	//		println("debugShowAddrs", s, sem, title)
	//	}

	//	if s != nil {
	//		linehead := ""
	//		for i := 0; i <= depth; i++ {
	//			linehead += "  "
	//		}
	//		println(linehead, s.elem, s.ticket, s.priority, title)
	//		if s.elem == unsafe.Pointer(sem) {
	//			println(linehead, "====have find it", s.elem, s.ticket, s.priority, title)
	//		}
	//		root.debugShowAddrs(s.prev, sem, title, depth+1)
	//		root.debugShowAddrs(s.next, sem, title, depth+1)
	//	}
}

// queue adds s to the blocked goroutines in semaRoot with priority.
// refer semaRoot.queue
func (root *semaRoot) queuePriority(addr *uint32, s *sudog, priority priorityType) {
	if false { //debug refer
		root.queue(addr, s, false)
		root.dequeue(addr)
		sync_runtime_SemacquireMutex(addr, false)
		sync_runtime_Semrelease(addr, false)
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
			if priority == priorityFirst || priority <= t.priority {
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
				for p.waitlink != nil && priority > p.waitlink.priority { //find the suitable node to insert after
					p = p.waitlink
				}
				//insert s after p
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
	//
	// s.ticket compared with zero in couple of places, therefore set lowest bit.
	// It will not affect treap's quality noticeably.
	s.ticket = fastrand() | 1
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

func (root *semaRoot) dequeuePriority(addr *uint32, priority priorityType) (num int32) {
	if false { //debug refer
		root.queue(addr, nil, false)
		root.dequeue(addr)
		sync_runtime_SemacquireMutex(addr, false)
		sync_runtime_Semrelease(addr, false)
		schedule()
	}

	ps := &root.treap
	s := *ps
	for ; s != nil; s = *ps {
		if s.elem == unsafe.Pointer(addr) {
			goto Found
		}
		if uintptr(unsafe.Pointer(addr)) < uintptr(s.elem) {
			ps = &s.prev
		} else {
			ps = &s.next
		}
	}
	return 0 //do not found

Found:
	saveHead := *s
	now := int64(0)
	if s.acquiretime != 0 {
		now = cputicks()
	}
	p, n := s, s.waitlink
	for ; p != nil && p.priority <= priority; p = n {
		n = p.waitlink
		num++
		casgstatus(p.g, _Gwaiting, _Grunnable)
		globrunqputhead(p.g)
		//println("====wakeup", addr, priority, p.g.goid, p.priority)
		p.priority = 0
		p.parent = nil
		p.elem = nil
		p.next = nil
		p.prev = nil
		p.ticket = 0
		p.waitlink = nil
		p.waittail = nil
	}

	if p != s {
		if t := p; t != nil {
			// Substitute t, also waiting on addr, for s in root tree of unique addrs.
			*ps = t
			t.ticket = saveHead.ticket
			t.parent = saveHead.parent
			if t.prev = saveHead.prev; t.prev != nil {
				t.prev.parent = t
			}
			if t.next = saveHead.next; t.next != nil {
				t.next.parent = t
			}
			if t.waitlink != nil {
				t.waittail = saveHead.waittail
			} else {
				t.waittail = nil
			}
			t.acquiretime = now
		} else {
			*s = saveHead //restore s temply
			// Rotate s down to be leaf of tree for removal, respecting priorities.
			for s.next != nil || s.prev != nil {
				if s.next == nil || s.prev != nil && s.prev.ticket < s.next.ticket {
					root.rotateRight(s)
				} else {
					root.rotateLeft(s)
				}
			}
			// Remove s, now a leaf.
			if s.parent != nil {
				if s.parent.prev == s {
					s.parent.prev = nil
				} else {
					s.parent.next = nil
				}
			} else {
				//println("set treap nil", root, addr)
				root.treap = nil
			}
		}
		s.parent = nil
		s.elem = nil
		s.next = nil
		s.prev = nil
		s.ticket = 0
		s.priority = 0
		s.waitlink = nil
		s.waittail = nil
	}

	return
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

//func (root *semaRoot) dequeuesudog(sem *uint32, su *sudog) {
//	ps := &root.treap
//	s := *ps
//	for ; s != nil; s = *ps {
//		if s.elem == unsafe.Pointer(sem) {
//			goto Found
//		}
//		if uintptr(unsafe.Pointer(sem)) < uintptr(s.elem) {
//			ps = &s.prev
//		} else {
//			ps = &s.next
//		}
//	}

//	throw("do not found") //do not found

//Found:
//	p := s
//	for p.waitlink != nil && p.waitlink != su {
//		p = p.waitlink
//	}
//	if p.waitlink == nil {
//		throw("do not found")
//	}
//	del := p.waitlink
//	p.waitlink = del.waitlink
//	del.waitlink = nil
//}

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

//type blockCheck struct {
//	sem   *uint32
//	s     *sudog
//	check func() bool
//}

//func (bc *blockCheck) init(sem *uint32, s *sudog, check func() bool) *blockCheck {
//	bc.sem = sem
//	bc.s = s
//	bc.check = check
//	return bc
//}

////do not use pointer receiver to escape to heap
//func (bc blockCheck) checkunlock(g_ *g, lock unsafe.Pointer) bool {
//	noblock := bc.check()
//	if noblock { //do not block, dequeue bc.s
//		root := semroot(bc.sem)
//		root.dequeuesudog(bc.sem, bc.s)
//	}
//	unlock((*mutex)(lock))
//	return !noblock
//}

//func goparkunlockWaitList(lock *mutex, reason string, traceEv byte, traceskip int) {
//	if false { //debug refer
//		goparkunlock(lock, "syncWaitList", traceEvGoBlockWaitList, 4)
//	}
//	//gopark(checkunlock, unsafe.Pointer(lock), reason, traceEv, traceskip)
//	unlock(lock)
//	gopark(nil, nil, reason, traceEv, traceskip)
//}
