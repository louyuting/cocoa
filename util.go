package cocoa

import (
	"time"
	"unsafe"
)

const (
	PtrSize = 4 << (^uintptr(0) >> 63)
)

// SliceHeader is a safe version of SliceHeader used within this project.
type SliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

// StringHeader is a safe version of StringHeader used within this project.
type StringHeader struct {
	Data unsafe.Pointer
	Len  int
}

const UnixTimeUnitOffset = uint64(time.Millisecond / time.Nanosecond)

// Returns the current Unix timestamp in milliseconds.
func CurrentTimeMillis() uint64 {
	return uint64(time.Now().UnixNano()) / UnixTimeUnitOffset
}

func ceilingPowerOfTwo(s int) int {
	n := s - 1
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	if n < 0 {
		return 1
	}
	return n + 1
}

func bitCount(i uint64) int {
	// HD, Figure 5-14
	i = i - ((i >> 1) & 0x5555555555555555)
	i = (i & 0x3333333333333333) + ((i >> 2) & 0x3333333333333333)
	i = (i + (i >> 4)) & 0x0f0f0f0f0f0f0f0f
	i = i + (i >> 8)
	i = i + (i >> 16)
	i = i + (i >> 32)
	return int(i & 0x7f)
}
