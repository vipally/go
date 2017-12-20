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
	buffLen = 10
	rCnt    = 2
	wCnt    = 2
)

var (
	ringBuffer = NewRingBuffer(buffLen)
	wg         WaitGroup
	lock       RWMutex
	isDebug    = true
)

func init() {
	cpuN := runtime.NumCPU()
	fmt.Println("GOMAXPROCS", cpuN)
	runtime.GOMAXPROCS(1)
}

func TestCycleBuffer(t *testing.T) {
	start := time.Now()
	doTest(t, dataCnt, rCnt, wCnt, buffLen, true, false)
	dur := time.Now().Sub(start)
	fmt.Printf("[lockfree]cost time %s for %d data buffLen=%d w/r=%d,%d avg=%s\n", dur, dataCnt, buffLen, wCnt, rCnt, dur/time.Duration(dataCnt))
}

func TestCycleBufferLock(t *testing.T) {
	start := time.Now()
	doTest(t, dataCnt, rCnt, wCnt, buffLen, true, true)
	dur := time.Now().Sub(start)
	fmt.Printf("[lock]cost time %s for %d data buffLen=%d w/r=%d,%d avg=%s\n", dur, dataCnt, buffLen, wCnt, rCnt, dur/time.Duration(dataCnt))
}

func doTest(t *testing.T, dataN, r, w int, _buffLen uint32, sync bool, lock bool) {
	if sync {
		ringBuffer.Init(buffLen)
		_w := w
		_r := r
		for i := 0; i < r+w; {
			if _w > 0 {
				wg.Add(1)
				go writerThread(i, dataN, w, lock)
				_w--
				i++
			}
			if _r > 0 {
				wg.Add(1)
				go readerThread(i+1, dataN, r, lock)
				_r--
				i++
			}
			//			if _w <= 0 && _r <= 0 {
			//				break
			//			}
		}
		//		for i := 0; i < r; i++ {
		//			wg.Add(1)
		//			go readerThread(i, dataN, r, lock)
		//		}
		//		for i := r; i < r+w; i++ {
		//			wg.Add(1)
		//			go writerThread(i, dataN, w, lock)
		//		}
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

/////////////////////////////////////

func Benchmark_100_50_10_10_lock(b *testing.B) {
	doBenchTest(b, 100, 10, 10, 50, true, true)
}
func Benchmark_100_20_10_10_lock(b *testing.B) {
	doBenchTest(b, 100, 10, 10, 20, true, true)
}
func Benchmark_100_5_10_10_lock(b *testing.B) {
	doBenchTest(b, 100, 10, 10, 5, true, true)
}
func Benchmark_100_20_2_2_lock(b *testing.B) {
	doBenchTest(b, 100, 2, 2, 20, true, true)
}
func Benchmark_100_20_5_5_lock(b *testing.B) {
	doBenchTest(b, 100, 5, 5, 20, true, true)
}
func Benchmark_10000_50_50_50_lock(b *testing.B) {
	doBenchTest(b, 10000, 50, 50, 50, true, true)
}

func doBenchTest(b *testing.B, dataN, r, w int, _buffLen int, sync bool, lock bool) {
	for i := 0; i < b.N; i++ {
		if sync {
			ringBuffer.Init(_buffLen)
			for i := 0; i < r; i++ {
				wg.Add(1)
				go readerThread(i, dataN, r, lock)
			}
			for i := 0; i < w; i++ {
				wg.Add(1)
				go writerThread(i, dataN, w, lock)
			}
			wg.Wait()
		} else {
			for i := 0; i < r; i++ {
				readerThread(i, dataN, r, lock)
			}
			for i := r; i < r+w; i++ {
				writerThread(i, dataN, w, lock)
			}
		}
	}
}

func delay() {
	t := 1000000
	m := 1
	for i := 1; i < t; i++ {
		m *= i
	}
	m = m
}

func readerThread(id int, dataN, workerN int, lock bool) {

	println("create readerThread", id, " ", runtime.GetGoroutineId())
	id = int(runtime.GetGoroutineId())

	if dataN%workerN != 0 {
		panic("")
	}

	if lock {
		readerThreadLock(id, dataN, workerN)
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
		//time.Sleep(time.Nanosecond)
	}
	println("readerThread end", id, " ", runtime.GetGoroutineId())
}

func writerThread(id int, dataN, workerN int, lock bool) {

	println("create writerThread", id, " ", runtime.GetGoroutineId())
	id = int(runtime.GetGoroutineId())

	if dataN%workerN != 0 {
		panic("")
	}

	if lock {
		writerThreadLock(id, dataN, workerN)
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

		//time.Sleep(time.Nanosecond)
		if isDebug {
			fmt.Printf("[%02d]writerThread CommitW %d %#v\n", id, i, ringBuffer)
		}
		ringBuffer.CommitW(idx)
		if isDebug {
			fmt.Printf("[%02d]Writer commit %03d %#v\n", id, idx, ringBuffer)
		}
		//time.Sleep(time.Nanosecond)
	}
	println("writerThread end", id, " ", runtime.GetGoroutineId())
}

func readerThreadLock(id int, dataN, workerN int) {
	num := dataN / workerN
	for i := 0; i < num; i++ {
		lock.RLock()
		//time.Sleep(time.Nanosecond)
		lock.RUnlock()
	}
	wg.Done()
}

func writerThreadLock(id int, dataN, workerN int) {
	num := dataN / workerN
	for i := 0; i < num; i++ {
		lock.Lock()
		//time.Sleep(time.Nanosecond)
		lock.Unlock()
	}
	wg.Done()
}
