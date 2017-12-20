package sync

//PriorityType is the type of priority in wait list.
type PriorityType = uint64

const (
	PriorityFirst PriorityType = 0
	PriorityLast  PriorityType = 1<<64 - 1
)

//Provided by runtime.
func runtime_goWaitWithPriority(waitSem *uint32, priority PriorityType)
func runtime_goAwakeWithPriority(awakeSem *uint32, priority PriorityType)

// WaitList block a list of goroutins which are waiting the same event.
// Every waiter has a priority.And the wake up event will wake up sleeper by priority.
type WaitList struct {
	//state int32
	sema uint32
}

// Wait push current goroutin in wait list order by priority.
// Which will stop current g and runs gschedule.
func (wl *WaitList) Wait(priority PriorityType) {
	runtime_goWaitWithPriority(&wl.sema, priority)
}

// Wakeup wakes up gotoutins that holds pri <= priority.
// Current goroutin will continue after that.
func (wl *WaitList) Wakeup(priority PriorityType) {
	runtime_goAwakeWithPriority(&wl.sema, priority)
}
