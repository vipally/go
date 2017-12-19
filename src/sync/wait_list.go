package sync

//PriorityType is the type of priority in wait list.
type PriorityType = int32

// WaitList block a list of goroutins which are waiting the same event.
// Every waiter has a priority.And the wake up event will wake up sleeper by priority.
type WaitList struct {
	state int32
	sema  uint32
}

// Wait push current g in Wait list with priority.
// Which will stop current g and run gschedule()
func (wl *WaitList) Wait(priority PriorityType) {}

// Wakeup wakes up gs that holds pri <= priority.
// Current g will continue.
func (wl *WaitList) Wakeup(priority PriorityType) {}
