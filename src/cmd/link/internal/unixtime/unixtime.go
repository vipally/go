// package unixtime implement unixtime.Now to get timestamp for linker.
// Which cannot refer runtime and time library.
package unixtime

// Monotonic times are reported as offsets from startNano.
// We initialize startNano to nanotime() - 1 so that on systems where
// monotonic time resolution is fairly low (e.g. Windows 2008
// which appears to have a default resolution of 15ms),
// we avoid ever reporting a nanotime of 0.
// (Callers may want to use 0 as "time not set".)
var startNano int64 = nanotime() - 1

func nanotime() int64
func now() (int64, int32)
func Now() int64 {
	sec, _ := now()
	return sec
}
