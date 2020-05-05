package cocoa

import (
	"math/rand"
	"sync/atomic"
	"unsafe"
)

type LocalCache interface {
	EnableEvict() bool

	Put(key []byte, value interface{})

	PutIfAbsent(key []byte, value interface{}) (prior interface{})

	Get(key []byte) (value interface{})

	Delete(key []byte)

	Contains(key []byte) (ok bool)

	Size() int
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

type PerformCleanupTask struct {
	cache *BoundedLocalCache
}

func (t *PerformCleanupTask) run() {
	if t.cache == nil {
		return
	}
	t.cache.maintenance()
}

const (
	//The number of attempts to insert into the write buffer before yielding.
	WriteBufferRetries = 100
)

type BoundedLocalCache struct {
	data ConcurrentMap

	windowDeque    *AccessOrderDeque
	probationDeque *AccessOrderDeque
	protectedDeque *AccessOrderDeque

	weightedSize int
	maximum      int

	windowWeightedSize int
	windowMaximum      int

	mainProtectedMaximum      int
	mainProtectedWeightedSize int

	readBuffer  *RingBuffer
	writeBuffer *RingBuffer

	evictable AtomicBool

	sketch *FrequencySketch
	//
	evictExecChan chan PerformCleanupTask
	drainStatus   *DrainStatus
}

// enableEvict returns if the cache evicts entries due to a maximum size or weight threshold.
func (c *BoundedLocalCache) EnableEvict() bool {
	return c.evictable.Get()
}

func (c *BoundedLocalCache) Put(key []byte, value interface{}) {
	c.PutIfAbsent(key, value)
}

func (c *BoundedLocalCache) PutIfAbsent(key []byte, value interface{}) (prior interface{}) {
	prior = c.data.PutIfAbsent(key, value)
	if prior == nil {
		// absent, so add kv to cache
		c.afterWrite(&AddTask{
			c: c,
			node: &Node{
				Key:     key,
				Value:   value,
				weight:  1,
				prev:    nil,
				next:    nil,
				queueIn: Window,
			},
			weight: 1,
		})
	} else {
		// present, so update the kv to cache
		c.afterWrite(&UpdateTask{
			c: c,
			node: &Node{
				Key:     key,
				Value:   value,
				weight:  1,
				prev:    nil,
				next:    nil,
				queueIn: Window,
			},
			weightDiff: 0,
		})
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
	c.afterRead(node)
	return node.Value
}

func (c *BoundedLocalCache) Delete(key []byte) {
	existed := c.data.Remove(key)
	if !existed {
		return
	}
	c.afterWrite(&DeleteTask{
		c: c,
		node: &Node{
			Key:     key,
			Value:   nil,
			weight:  1,
			prev:    nil,
			next:    nil,
			queueIn: Window,
		},
	})
}

func (c *BoundedLocalCache) Contains(key []byte) (ok bool) {
	return c.data.Contains(key)
}

func (c *BoundedLocalCache) Size() int {
	return c.data.Len()
}

func (c *BoundedLocalCache) performCleanUp(t task) {
	c.evictExecChan <- PerformCleanupTask{
		cache: c,
	}
	t.run()
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
	c.readBuffer.DrainTo(c.onAccess)
}

func (c *BoundedLocalCache) drainWriteBuffer() {
	c.writeBuffer.DrainTo(c.onWrite)
	c.drainStatus.set(ProcessingToRequired)
}

func (c *BoundedLocalCache) evictEntries() {
	if !c.EnableEvict() {
		return
	}
	candidates := c.evictFromWindow()
	c.evictFromMain(candidates)
}

//  Attempts to evict the entry. A removal due to size may be ignored if the entry was updated and is no longer eligible for eviction.
func (c *BoundedLocalCache) evictEntry(node *Node) {
	if node == nil || len(node.Key) == 0 {
		return
	}
	c.data.Remove(node.Key)
	if node.inWindow() {
		c.windowDeque.Remove(node)
	} else if node.inMainProbation() {
		c.probationDeque.Remove(node)
	} else {
		c.protectedDeque.Remove(node)
	}
	return
}

// Evicts entries from the window space into the main space while the window size exceeds a maximum.
// return the number of candidate entries evicted from the window space
func (c *BoundedLocalCache) evictFromWindow() int {
	candidates := 0
	node := c.accessOrderWindowDeque().GetFront()
	for c.windowWeightedSize > c.windowMaximum {
		if node == nil {
			break
		}
		next := node.next
		c.accessOrderWindowDeque().Remove(node)
		c.accessOrderProbationDeque().PushBack(node)
		candidates++
		c.windowWeightedSize = c.windowWeightedSize - node.weight
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
	victim := c.accessOrderProbationDeque().GetFront()
	candidate := c.accessOrderProbationDeque().GetBack()

	for c.weightedSize > c.maximum {
		// Stop trying to evict candidates and always prefer the victim
		if candidates == 0 {
			candidate = nil
		}
		// Try evicting from the protected and window queues
		if candidate == nil && victim == nil {
			if victimQueue == Probation {
				victim = c.accessOrderProtectedDeque().GetFront()
				victimQueue = Protected
				continue
			} else if victimQueue == Protected {
				victim = c.accessOrderWindowDeque().GetFront()
				victimQueue = Window
				continue
			}
			break
		}

		// Evict immediately if only one of the entries is present
		if victim == nil {
			previous := candidate.prev
			evict := candidate
			candidate = previous
			candidates--
			c.evictEntry(evict)
			continue
		} else if candidate == nil {
			evict := victim
			victim = victim.next
			c.evictEntry(evict)
			continue
		}

		// Evict immediately if an entry was collected
		victimKey := victim.Key
		candidateKey := candidate.Key
		if len(victimKey) == 0 {
			evict := victim
			victim = victim.next
			c.evictEntry(evict)
			continue
		} else if len(candidateKey) == 0 {
			candidates--
			evict := candidate
			candidate = candidate.prev
			c.evictEntry(evict)
			continue
		}
		if candidate.weight > c.maximum {
			candidates--
			evict := candidate
			candidate = candidate.prev
			c.evictEntry(evict)
			continue
		}
		// Evict the entry with the lowest frequency
		candidates--
		if c.admit(candidateKey, victimKey) {
			evict := victim
			victim = victim.next
			c.evictEntry(evict)
			candidate = candidate.prev
		} else {
			evict := candidate
			candidate = candidate.prev
			c.evictEntry(evict)
		}
	}
	return
}

// Determines if the candidate should be accepted into the main space, as determined by its
// frequency relative to the victim. A small amount of randomness is used to protect against hash
// collision attacks, where the victim's frequency is artificially raised so that no new entries
// are admitted.
// return if the candidate should be admitted and the victim ejected
func (c *BoundedLocalCache) admit(candidateKey []byte, victimKey []byte) bool {
	candidateFreq := c.sketch.frequency(candidateKey)
	victimFreq := c.sketch.frequency(victimKey)
	if candidateFreq > victimFreq {
		return true
	} else if candidateFreq <= 5 {
		// candidateFreq<=victimFreq && candidateFreq <= 5
		return false
	}
	random := rand.Int()
	return (random & 127) == 0
}

func (c *BoundedLocalCache) onAccess(p unsafe.Pointer) {
	panic("implement me")
}

func (c *BoundedLocalCache) onWrite(p unsafe.Pointer) {
	if p == nil {
		return
	}
	taskPtr := (*task)(p)
	if taskPtr == nil {
		return
	}
	task := *taskPtr
	task.run()
}

//======================================================================================================================
func (c *BoundedLocalCache) accessOrderWindowDeque() *AccessOrderDeque {
	return c.windowDeque
}
func (c *BoundedLocalCache) accessOrderProbationDeque() *AccessOrderDeque {
	return c.probationDeque
}
func (c *BoundedLocalCache) accessOrderProtectedDeque() *AccessOrderDeque {
	return c.protectedDeque
}

func (c *BoundedLocalCache) afterRead(node *Node) {
	// Might lose some read record if readBuffer.Offer return Failed
	delayable := c.readBuffer.Offer(unsafe.Pointer(node)) != Full
	if c.shouldDrainBuffers(delayable) {
		c.scheduleDrainBuffers()
	}
}

func (c *BoundedLocalCache) afterWrite(t task) {
	for i := 0; i < WriteBufferRetries; i++ {
		if c.writeBuffer.Offer(unsafe.Pointer(&t)) == Success {
			c.scheduleAfterWrite()
			return
		}
		c.scheduleDrainBuffers()
	}

	// perform task directly
	c.performCleanUp(t)
}

func (c *BoundedLocalCache) shouldDrainBuffers(delayable bool) bool {
	switch c.drainStatus.get() {
	case Idle:
		return !delayable
	case Required:
		return true
	case ProcessingToIdle:
		return false
	case ProcessingToRequired:
		return false
	default:
		panic("implement me")
	}
}

func (c *BoundedLocalCache) scheduleAfterWrite() {
	status := c.drainStatus
	for {
		switch status.get() {
		case Idle:
			status.casDrainStatus(Idle, Required)
			c.scheduleDrainBuffers()
			return
		case Required:
			c.scheduleDrainBuffers()
			return
		case ProcessingToIdle:
			if status.casDrainStatus(ProcessingToIdle, ProcessingToRequired) {
				return
			}
			continue
		case ProcessingToRequired:
			return
		default:
			panic("implement me")
		}
	}
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
