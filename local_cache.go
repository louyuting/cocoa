package cocoa

import (
	"container/list"
	"sync/atomic"
	"unsafe"
)

type LocalCache interface {
	Put(key []byte, value interface{})

	PutIfAbsent(key []byte, value interface{}) (prior interface{})

	Get(key []byte) (value interface{})

	Delete(key []byte)

	Contains(key []byte) (ok bool)

	Size() int
}

type Node struct {
	Key   []byte
	Value interface{}
}

type DrainStatus int32

const (
	Idle                 DrainStatus = iota
	Required                         = 1
	ProcessingToIdle                 = 2
	ProcessingToRequired             = 3
)

func (s *DrainStatus) get() DrainStatus {
	statusPtr := (*int32)(unsafe.Pointer(s))
	return DrainStatus(atomic.LoadInt32(statusPtr))
}
func (s *DrainStatus) set(status DrainStatus) {
	statusPtr := (*int32)(unsafe.Pointer(s))
	newStatus := int32(status)
	atomic.StoreInt32(statusPtr, newStatus)
}

func (s *DrainStatus) casDrainStatus(expect DrainStatus, update DrainStatus) bool {
	statusPtr := (*int32)(unsafe.Pointer(s))
	oldStatus := int32(expect)
	newStatus := int32(update)
	return atomic.CompareAndSwapInt32(statusPtr, oldStatus, newStatus)
}

type QueueType int32

const (
	Window QueueType = iota
	Probation
	Protected
)

type PerformCleanupTask struct {
	cache *BoundedLocalCache
}

func (t *PerformCleanupTask) run() {
	if t.cache == nil {
		return
	}
	t.cache.maintenance()
}

type BoundedLocalCache struct {
	data *SafeMap

	readBuffer  *RingBuffer
	writeBuffer *RingBuffer
	//
	evictExecChan chan PerformCleanupTask
	drainStatus   *DrainStatus
}

func (c *BoundedLocalCache) Put(key []byte, value interface{}) {
	c.PutIfAbsent(key, value)
}

func (c *BoundedLocalCache) PutIfAbsent(key []byte, value interface{}) (prior interface{}) {
	prior = c.data.PutIfAbsent(key, value)
	if prior == nil {
		// absent, so add kv to cache
		c.afterWrite(&AddTask{})
	} else {
		// present, so update the kv to cache
		c.afterWrite(&UpdateTask{})
	}
	return prior
}

func (c *BoundedLocalCache) Get(key []byte) (value interface{}) {
	val, existed := c.data.Get(key)
	if !existed {
		return nil
	}
	node, existed := val.(*Node)
	if !existed {
		return nil
	}
	now := CurrentTimeMillis()
	c.afterRead(node, now)
	return node.Value
}

func (c *BoundedLocalCache) Delete(key []byte) {
	existed := c.data.Remove(key)
	if !existed {
		return
	}
	c.afterWrite(&DeleteTask{})
}

func (c *BoundedLocalCache) Contains(key []byte) (ok bool) {
	return c.data.Contains(key)
}

func (c *BoundedLocalCache) Size() int {
	return c.data.Len()
}

func (c *BoundedLocalCache) performCleanUp() {

}

//======================================================================================================================
// Performs the pending maintenance work and sets the state flags during processing to avoid
// excess scheduling attempts. The read buffer and write buffer are drained,
// followed by expiration, and size-based eviction.
func (c *BoundedLocalCache) maintenance() {
	c.drainStatus.set(ProcessingToIdle)
	defer func() {
		if (c.drainStatus.get() != ProcessingToIdle) || !c.drainStatus.casDrainStatus(ProcessingToIdle, Idle) {
			c.drainStatus.set(Required)
		}
	}()

	c.drainReadBuffer()
	c.drainWriteBuffer()

	c.evictEntries()
}

func (c *BoundedLocalCache) drainReadBuffer() {

}

func (c *BoundedLocalCache) drainWriteBuffer() {

}

// enableEvict returns if the cache evicts entries due to a maximum size or weight threshold.
func (c *BoundedLocalCache) enableEvict() bool {
	// TODO
	return true
}
func (c *BoundedLocalCache) evictEntries() {
	if !c.enableEvict() {
		return
	}
	candidates := c.evictFromWindow()
	c.evictFromMain(candidates)
}

//  Attempts to evict the entry. A removal due to size may be ignored if the entry was updated and is no longer eligible for eviction.
func (c *BoundedLocalCache) evictEntry(node *Node) {

}

// Evicts entries from the window space into the main space while the window size exceeds a maximum.
// return the number of candidate entries evicted from the window space
func (c *BoundedLocalCache) evictFromWindow() int {
	candidates := 0
	node := c.accessOrderWindowDeque().Front()
	for c.windowWeightedSize() > c.windowMaximum() {
		if node == nil {
			break
		}
		next := node.Next()
		c.accessOrderWindowDeque().Remove(node)
		c.accessOrderProbationDeque().PushBack(node.Value)
		candidates++
		c.setWindowWeightedSize(c.windowWeightedSize() - 1)
		node = next
	}
	return candidates
}

//	Evicts entries from the main space if the cache exceeds the maximum capacity. The main space
//  determines whether admitting an entry (coming from the window space) is preferable to retaining
//  the eviction policy's victim. This is decision is made using a frequency filter so that the
//  least frequently used entry is removed.
//
// The window space candidates were previously placed in the MRU position and the eviction
// policy's victim is at the LRU position. The two ends of the queue are evaluated while an
// eviction is required. The number of remaining candidates is provided and decremented on
// eviction, so that when there are no more candidates the victim is evicted.
func (c *BoundedLocalCache) evictFromMain(candidates int) {
	victimQueue := Probation
	victim := c.accessOrderProbationDeque().Front()
	candidate := c.accessOrderProbationDeque().Back()

	for c.weightedSize() > c.maximum() {
		// Stop trying to evict candidates and always prefer the victim
		if candidates == 0 {
			candidate = nil
		}
		// Try evicting from the protected and window queues
		if candidate == nil && victim == nil {
			if victimQueue == Probation {
				victim = c.accessOrderProtectedDeque().Front()
				victimQueue = Protected
				continue
			} else if victimQueue == Protected {
				victim = c.accessOrderWindowDeque().Front()
				victimQueue = Window
				continue
			}
			break
		}

		//
	}
	return
}

//======================================================================================================================
func (c *BoundedLocalCache) accessOrderWindowDeque() *list.List {
	panic("implement me")
}
func (c *BoundedLocalCache) accessOrderProbationDeque() *list.List {
	panic("implement me")
}
func (c *BoundedLocalCache) accessOrderProtectedDeque() *list.List {
	panic("implement me")
}
func (c *BoundedLocalCache) windowWeightedSize() int {
	panic("implement me")
}

func (c *BoundedLocalCache) setWindowWeightedSize(windowSize int) {
	panic("implement me")
}

func (c *BoundedLocalCache) windowMaximum() int {
	panic("implement me")
}

// Returns the combined weight of the values in the cache (may be negative)
func (c *BoundedLocalCache) weightedSize() int {
	panic("implement me")
}

//Returns the maximum weighted size.
func (c *BoundedLocalCache) maximum() int {
	panic("implement me")
}

func (c *BoundedLocalCache) afterRead(node *Node, now uint64) {

}

func (c *BoundedLocalCache) afterWrite(t task) {

}

func (c *BoundedLocalCache) scheduleDrainBuffers() {
	status := c.drainStatus.get()
	if status >= ProcessingToIdle {
		// processing, return directly.
		return
	}
	if status.casDrainStatus(status, ProcessingToIdle) {
		c.evictExecChan <- PerformCleanupTask{cache: c}
	}
}

// async to clean up cache
// Caller must guarantee only one async goroutine to execute this function.
func (c *BoundedLocalCache) asyncCleanUp() {
	for {
		task := <-c.evictExecChan
		task.run()
		if c.drainStatus.get() == Required {
			c.scheduleDrainBuffers()
		}
	}
}
