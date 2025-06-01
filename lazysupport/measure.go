package lazysupport

import (
	"fmt"
	"runtime"
	"time"
)

type measurement struct {
	elapsed        time.Duration
	allocations    uint64
	bytesAllocated databytes
}

type databytes uint64

func (b databytes) String() string {
	switch {
	case b < 1024:
		return fmt.Sprintf("%d bytes", b)
	case b < 1024*1024:
		return fmt.Sprintf("%d kb", b/1024)
	case b < 1024*1024*1024:
		return fmt.Sprintf("%d mb", b/(1024*1024))
	default:
		return fmt.Sprintf("%d gb", b/(1024*1024*1024))
	}
}

func (m *measurement) String() string {
	return fmt.Sprintf("%s %d allocs (%s)", m.elapsed, m.allocations, m.bytesAllocated)
}

func MeasureText[T any](name string, fn func() T) T {
	m := &runtime.MemStats{}
	m2 := &runtime.MemStats{}
	start := time.Now()
	var done bool
	notes := ""
	defer func() {
		runtime.ReadMemStats(m2)
		if !done {
			notes = "⚠"
		}
		fmt.Printf("%s %s %d (%s)%s\n", name, time.Since(start), m2.Mallocs-m.Mallocs, databytes(m2.TotalAlloc-m.TotalAlloc), notes)
	}()
	runtime.ReadMemStats(m)
	v := fn()
	done = true
	return v

}
