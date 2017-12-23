package sync_test

import (
	"time"

	"fmt"
	"runtime"
	. "sync"
	"testing"
)

const (
	dataCnt = 100
	buffLen = 20
	rCnt    = 2
	wCnt    = 2
	delayT  = 5000
)

var (
	ringBuffer = NewRingBuffer(buffLen)
	wg         WaitGroup
	lock       RWMutex

	isDebug = true
)

func init() {
	cpuN := runtime.NumCPU()
	//fmt.Println("GOMAXPROCS", cpuN)
	runtime.GOMAXPROCS(cpuN)
}

func TestRingBuffer(t *testing.T) {
	if true {
		goMAXPROCS := runtime.GOMAXPROCS(0)
		start := time.Now()
		doTest(t, dataCnt, rCnt, wCnt, buffLen, true, false)
		dur := time.Now().Sub(start)
		fmt.Printf("[RingBuffer ]cost time % 12s for %d data, w=%d r=%d avg=%s buffLen=%d GOMAXPROCS=%d\n", dur, dataCnt, wCnt, rCnt, dur/time.Duration(dataCnt), buffLen, goMAXPROCS)
	}

	if true {
		start := time.Now()
		doTest(t, dataCnt, rCnt, wCnt, buffLen, true, true)
		dur := time.Now().Sub(start)
		fmt.Printf("[MutexBuffer]cost time % 12s for %d data, w=%d r=%d avg=%s\n", dur, dataCnt, wCnt, rCnt, dur/time.Duration(dataCnt))
	}
}

func doTest(t *testing.T, dataN, r, w int, _buffLen uint32, sync bool, lock bool) {
	if sync {
		ringBuffer.Init(buffLen)
		for i := 0; i < r; i++ {
			wg.Add(1)
			go readeWorker(i, dataN, r, lock)
		}
		for i := r; i < r+w; i++ {
			wg.Add(1)
			go writeWorker(i, dataN, w, lock)
		}
		wg.Wait()
	} else {
	}
}

func Benchmark_100_50_10_10(b *testing.B) {
	doBenchTest(b, 100, 10, 10, 50, true, false)
}

func Benchmark_100_20_10_10(b *testing.B) {
	doBenchTest(b, 100, 10, 10, 20, true, false)
}

func Benchmark_100_5_10_10(b *testing.B) {
	doBenchTest(b, 100, 10, 10, 5, true, false)
}
func Benchmark_100_20_2_2(b *testing.B) {
	doBenchTest(b, 100, 2, 2, 20, true, false)
}
func Benchmark_100_20_5_5(b *testing.B) {
	doBenchTest(b, 100, 5, 5, 20, true, false)
}

func Benchmark_10000_50_50_50(b *testing.B) {
	doBenchTest(b, 10000, 50, 50, 50, true, false)
}

///////////////////////////////////////

//func Benchmark_100_50_10_10_lock(b *testing.B) {
//	doBenchTest(b, 100, 10, 10, 50, true, true)
//}
//func Benchmark_100_20_10_10_lock(b *testing.B) {
//	doBenchTest(b, 100, 10, 10, 20, true, true)
//}
//func Benchmark_100_5_10_10_lock(b *testing.B) {
//	doBenchTest(b, 100, 10, 10, 5, true, true)
//}
//func Benchmark_100_20_2_2_lock(b *testing.B) {
//	doBenchTest(b, 100, 2, 2, 20, true, true)
//}
//func Benchmark_100_20_5_5_lock(b *testing.B) {
//	doBenchTest(b, 100, 5, 5, 20, true, true)
//}
//func Benchmark_10000_50_50_50_lock(b *testing.B) {
//	doBenchTest(b, 10000, 50, 50, 50, true, true)
//}

func doBenchTest(b *testing.B, dataN, r, w int, _buffLen int, sync bool, lock bool) {
	for i := 0; i < b.N; i++ {
		if sync {
			ringBuffer.Init(_buffLen)
			for i := 0; i < r; i++ {
				wg.Add(1)
				go readeWorker(i, dataN, r, lock)
			}
			for i := 0; i < w; i++ {
				wg.Add(1)
				go writeWorker(i, dataN, w, lock)
			}
			wg.Wait()
		} else {
			for i := 0; i < r; i++ {
				readeWorker(i, dataN, r, lock)
			}
			for i := r; i < r+w; i++ {
				writeWorker(i, dataN, w, lock)
			}
		}
	}
}

func delay() {
	t := delayT
	m := 1
	for i := 1; i <= t; i++ {
		for j := 1; j <= t; j++ {
			m *= j
		}
	}
	m = m
}

func readeWorker(id int, dataN, workerN int, lock bool) {
	if isDebug {
		println("create readerThread", id, " ", runtime.GetGoroutineId())
	}
	id = int(runtime.GetGoroutineId())

	if dataN%workerN != 0 {
		panic("")
	}

	if lock {
		readWorkerMutex(id, dataN, workerN)
		//wg.Done()
		return
	}
	defer wg.Done()
	num := dataN / workerN
	for i := 1; i <= num; i++ {
		if isDebug {
			fmt.Printf("readerThread[%02d] ReserveR %d/%d\n", id, i, num)
		}
		idx := ringBuffer.ReserveR()
		if isDebug {
			fmt.Printf("[%02d]Reader read %03d wait %#v\n", id, idx, ringBuffer)
		}
		delay()
		ringBuffer.CommitR(idx)
		if isDebug {
			fmt.Printf("[%02d]Reader CommitR %03d  %#v\n", id, idx, ringBuffer)
		}
	}
	if isDebug {
		println("readerThread end", id, " ", runtime.GetGoroutineId())
	}
}

func writeWorker(id int, dataN, workerN int, lock bool) {
	if isDebug {
		println("create writerThread", id, " ", runtime.GetGoroutineId())
	}
	id = int(runtime.GetGoroutineId())

	if dataN%workerN != 0 {
		panic("")
	}

	if lock {
		writeWorkerMutex(id, dataN, workerN)
		return
	}
	defer wg.Done()
	num := dataN / workerN
	for i := 1; i <= num; i++ {
		if isDebug {
			fmt.Printf("writerThread[%02d] ReserveW %d/%d\n", id, i, num)
		}
		idx := ringBuffer.ReserveW()
		if isDebug {
			fmt.Printf("[%02d]Writer write %03d  %#v\n", id, idx, ringBuffer)
		}

		delay()

		if isDebug {
			fmt.Printf("[%02d]writerThread CommitW %d %#v\n", id, i, ringBuffer)
		}
		ringBuffer.CommitW(idx)
		if isDebug {
			fmt.Printf("[%02d]Writer commit %03d %#v\n", id, idx, ringBuffer)
		}
	}
	if isDebug {
		println("writerThread end", id, " ", runtime.GetGoroutineId())
	}
}

func readWorkerMutex(id int, dataN, workerN int) {
	num := dataN / workerN
	for i := 0; i < num; i++ {
		lock.RLock()
		delay()
		lock.RUnlock()
	}
	wg.Done()
}

func writeWorkerMutex(id int, dataN, workerN int) {
	num := dataN / workerN
	for i := 0; i < num; i++ {
		lock.Lock()
		delay()
		lock.Unlock()
	}
	wg.Done()
}
