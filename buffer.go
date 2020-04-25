package cocoa

import "sync/atomic"

const (
	BufferSize = 16
	// Assume 4-byte references and 64-byte cache line (16 elements per line)
	SpaceSize = BufferSize << 4
	SpaceMask = SpaceSize - 1
	Offset    = 16
)

type BufferStatus uint8

const (
	Full    BufferStatus = iota
	Success              = 1
	Failed               = 2
)

type ElemConsumer func(*Node)

type RingBuffer struct {
	buf *atomicArray
	r   uint32
	w   uint32
}

func NewRingBuffer() *RingBuffer {
	ret := &RingBuffer{
		buf: newAtomicArray(SpaceSize),
		r:   0,
		w:   0,
	}
	return ret
}

// Offer inserts the specified element into this buffer if it is possible to do so immediately without
// violating capacity restrictions. The addition is allowed to fail spuriously if multiple
// threads insert concurrently.
func (r *RingBuffer) Offer(n *Node) BufferStatus {
	// read index
	head := atomic.LoadUint32(&r.r)
	// write index
	tail := atomic.LoadUint32(&r.w)
	size := tail - head
	if size >= SpaceSize {
		return Full
	}
	if atomic.CompareAndSwapUint32(&r.w, tail, tail+Offset) {
		idx := int(tail & SpaceMask)
		r.buf.set(idx, n)
		return Success
	}
	return Failed
}

// Drains the buffer, sending each element to the consumer for processing.
// The caller must ensure that a consumer has exclusive read access to the buffer.
func (r *RingBuffer) DrainTo(consumer ElemConsumer) {
	// read index
	head := atomic.LoadUint32(&r.r)
	// write index
	tail := atomic.LoadUint32(&r.w)
	size := tail - head
	if size == 0 {
		return
	}

	for tail != head {
		idx := int(head & SpaceMask)
		e := r.buf.get(idx)
		if e == nil {
			// not published yet
			break
		}
		r.buf.set(idx, nil)
		consumer(e)
		head += Offset
	}
	atomic.StoreUint32(&r.r, head)
}
