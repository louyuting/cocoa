package cocoa

import (
	"sync/atomic"
	"unsafe"
)

const (
	// The max number of element this buffer could store.
	bufferSize = 16
	// 256
	spaceSize = bufferSize << 4
	spaceMask = spaceSize - 1
	offset    = 16
)

type bufferStatus uint8

const (
	full    bufferStatus = iota
	success              = 1
	failed               = 2
)

// ringBuffer is the multi producer and one consumer buffer.
// ringBuffer is a lock-free fixed-size multi-producer,
// single-consumer queue. The multi producers can offer from the tail.
// and single consumer can poll from the head.
type ringBuffer struct {
	buf *atomicArray
	r   uint32
	w   uint32
}

func newRingBuffer() *ringBuffer {
	ret := &ringBuffer{
		buf: newAtomicArray(spaceSize),
		r:   0,
		w:   0,
	}
	return ret
}

// offer inserts the pointer of specified element into this buffer if it is possible to do so immediately without
// violating capacity restrictions. The addition is allowed to fail spuriously if multiple threads insert concurrently.
func (r *ringBuffer) offer(n unsafe.Pointer) bufferStatus {
	// read index
	head := atomic.LoadUint32(&r.r)
	// write index
	tail := atomic.LoadUint32(&r.w)
	size := tail - head
	if size >= spaceSize {
		return full
	}
	if atomic.CompareAndSwapUint32(&r.w, tail, tail+offset) {
		idx := int(tail & spaceMask)
		r.buf.set(idx, n)
		return success
	}
	return failed
}

// poll removes and returns the element at the head of buffer. It returns nil if the
// queue is empty. It must only be called by a single consumer.
func (r *ringBuffer) poll() task {
	// read index
	head := atomic.LoadUint32(&r.r)
	// write index
	tail := atomic.LoadUint32(&r.w)
	size := tail - head
	if size == 0 {
		return nil
	}
	idx := int(head & spaceMask)
	e := r.buf.get(idx)
	if e == nil {
		// not published yet
		return nil
	}
	r.buf.set(idx, nil)
	tPtr := (*task)(e)
	t := *tPtr
	//t.run()
	head += offset
	atomic.StoreUint32(&r.r, head)
	return t
}

// Drains the buffer, sending each element to the consumer for processing.
// The caller must ensure that a consumer has exclusive read access to the buffer.
func (r *ringBuffer) drainBuf() {
	// read index
	head := atomic.LoadUint32(&r.r)
	// write index
	tail := atomic.LoadUint32(&r.w)
	size := tail - head
	if size == 0 {
		return
	}

	for tail != head {
		idx := int(head & spaceMask)
		e := r.buf.get(idx)
		if e == nil {
			// not published yet
			break
		}
		r.buf.set(idx, nil)
		tPtr := (*task)(e)
		t := *tPtr
		t.run()
		head += offset
	}
	atomic.StoreUint32(&r.r, head)
}
