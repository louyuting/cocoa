package cocoa

import (
	"fmt"
	"sync/atomic"
	"unsafe"
)

type atomicArray struct {
	// The base address for real data array
	base unsafe.Pointer
	// The length of slice(array), it is readonly
	length int
	data   []unsafe.Pointer
}

// New atomicArray with initializing field data
// len: length of array
func newAtomicArray(len int) *atomicArray {
	ret := &atomicArray{
		length: len,
		data:   make([]unsafe.Pointer, len, len),
	}
	// calculate base address for real data array
	sliHeader := (*SliceHeader)(unsafe.Pointer(&ret.data))
	ret.base = unsafe.Pointer((*unsafe.Pointer)(sliHeader.Data))
	return ret
}

func (a *atomicArray) elementOffset(idx int) unsafe.Pointer {
	if idx >= a.length && idx < 0 {
		panic(fmt.Sprintf("The index (%d) is out of bounds, length is %d.", idx, a.length))
	}
	basePtr := a.base
	return unsafe.Pointer(uintptr(basePtr) + uintptr(idx*PtrSize))
}

func (a *atomicArray) get(idx int) unsafe.Pointer {
	// a.elementOffset(idx) return the secondary pointer of real struct, which is the pointer to the a.data[idx]
	// then convert to (*unsafe.Pointer)
	return atomic.LoadPointer((*unsafe.Pointer)(a.elementOffset(idx)))
}

func (a *atomicArray) set(idx int, n unsafe.Pointer) {
	atomic.StorePointer((*unsafe.Pointer)(a.elementOffset(idx)), n)
}

func (a *atomicArray) compareAndSet(idx int, except, update unsafe.Pointer) bool {
	// a.elementOffset(idx) return the secondary pointer of real struct, which is the pointer to the a.data[idx]
	// then convert to (*unsafe.Pointer)
	// update secondary pointer
	return atomic.CompareAndSwapPointer((*unsafe.Pointer)(a.elementOffset(idx)), except, update)
}

func (a *atomicArray) len() int {
	return a.length
}
