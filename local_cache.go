package cocoa

import (
	"math/rand"
	"runtime"
	"sync/atomic"
	"unsafe"
)

//type LocalCache interface {
//	EnableEvict() bool
//
//	Put(key []byte, value interface{})
//
//	PutIfAbsent(key []byte, value interface{}) (prior interface{})
//
//	Get(key []byte) (value interface{})
//
//	Delete(key []byte)
//
//	Contains(key []byte) (ok bool)
//
//	Size() int
//}

type DrainState int32

/**




 */
const (
	Idle                 DrainState = iota
	Required                        = 1
	ProcessingToIdle                = 2
	ProcessingToRequired            = 3
)

func (s *DrainState) get() DrainState {
	statePtr := (*int32)(unsafe.Pointer(s))
	return DrainState(atomic.LoadInt32(statePtr))
}
func (s *DrainState) set(state DrainState) {
	statePtr := (*int32)(unsafe.Pointer(s))
	newState := int32(state)
	atomic.StoreInt32(statePtr, newState)
}

func (s *DrainState) casDrainStatus(expect DrainState, update DrainState) bool {
	statePtr := (*int32)(unsafe.Pointer(s))
	oldState := int32(expect)
	newState := int32(update)
	return atomic.CompareAndSwapInt32(statePtr, oldState, newState)
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

var (
	// The maximum capacity of the write buffer.
	WriteBufferMaxCapacity = 128 * ceilingPowerOfTwo(runtime.NumCPU())
)

const (
	//The number of attempts to insert into the write buffer before yielding.
	WriteBufferRetries = 100
)

// BoundedLocalCache is the local bounded cache. the eviction strategy supports
// 1. size-based eviction
//
//
type BoundedLocalCache struct {
	data *SegmentHashMap

	windowDeque    *AccessOrderDeque
	probationDeque *AccessOrderDeque
	protectedDeque *AccessOrderDeque

	// the cache weighted size
	weightedSize int
	maximum      int

	// the window deque weighted size
	windowWeightedSize int
	windowMaximum      int

	// the protected deque weighted size
	mainProtectedWeightedSize int
	mainProtectedMaximum      int

	readBuffer  *ringBuffer
	writeBuffer *ringBuffer

	enableEvict AtomicBool

	sketch *FrequencySketch
	//
	evictExecChan chan PerformCleanupTask
	drainState    *DrainState
}

// enableEvict returns if the cache evicts entries due to a maximum size or weight threshold.
func (c *BoundedLocalCache) EnableEvict() bool {
	return c.enableEvict.Get()
}

func (c *BoundedLocalCache) Put(key []byte, value interface{}) {
	if len(key) == 0 {
		return
	}
	seg := c.data.getSegment(c.data.hash(key))
	seg.mux.Lock()
	priorNode, existed := seg.data[*bytesToString(key)]
	if !existed {
		node := &Node{
			Key:     key,
			Value:   value,
			weight:  1,
			prev:    nil,
			next:    nil,
			dequeIn: Window,
		}
		seg.data[*bytesToString(key)] = node
		seg.mux.Unlock()
		c.afterWrite(&AddTask{
			c:      c,
			node:   node,
			weight: 1,
		})
	} else {
		priorNode.Value = value
		seg.mux.Unlock()
		c.afterWrite(&UpdateTask{
			c:          c,
			node:       priorNode,
			weightDiff: 0,
		})
	}
}

// PutIfAbsent put the key/value into cache if key don't exist in cache before;
// else return the prior value and do nothing
func (c *BoundedLocalCache) PutIfAbsent(key []byte, value interface{}) interface{} {
	if len(key) == 0 {
		panic("key is empty.")
	}
	seg := c.data.getSegment(c.data.hash(key))
	seg.mux.Lock()
	priorNode, existed := seg.data[*bytesToString(key)]
	if !existed {
		node := &Node{
			Key:     key,
			Value:   value,
			weight:  1,
			prev:    nil,
			next:    nil,
			dequeIn: Window,
		}
		seg.data[*bytesToString(key)] = node
		seg.mux.Unlock()
		c.afterWrite(&AddTask{
			c:      c,
			node:   node,
			weight: 1,
		})
		return nil
	} else {
		return priorNode
	}
}

func (c *BoundedLocalCache) Get(key []byte) (value interface{}) {
	node, existed := c.data.Get(key)
	if !existed {
		return nil
	}
	c.afterRead(node)
	return node.Value
}

func (c *BoundedLocalCache) Delete(key []byte) interface{} {
	prior := c.data.Remove(key)
	if prior == nil {
		return nil
	}
	c.afterWrite(&DeleteTask{
		c:    c,
		node: prior,
	})
	return prior.Value
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
	c.drainState.set(ProcessingToIdle)
	defer func() {
		// 1. after eviction, the status is not ProcessingToIdle, so need to continue drain buffer, mark as Required
		// 2. after eviction, the status is ProcessingToIdle, fail to cas update status to Idle, so need to continue drain buffer, mark as Required
		if (c.drainState.get() != ProcessingToIdle) || !c.drainState.casDrainStatus(ProcessingToIdle, Idle) {
			c.drainState.set(Required)
		}
	}()

	c.drainReadBuffer()
	c.drainWriteBuffer()

	c.evictEntries()
}

func (c *BoundedLocalCache) drainReadBuffer() {
	c.readBuffer.drainBuf()
}

func (c *BoundedLocalCache) drainWriteBuffer() {
	for i := 0; i < WriteBufferMaxCapacity; i++ {
		t := c.writeBuffer.poll()
		if t == nil {
			return
		}
		t.run()
	}
	// the write buffer has not been drained,
	// the state machine of drainState is switched to the ProcessingToRequired
	c.drainState.set(ProcessingToRequired)
}

func (c *BoundedLocalCache) evictEntries() {
	if !c.EnableEvict() {
		return
	}
	candidateNum := c.evictFromWindow()
	c.evictFromMain(candidateNum)
}

//  Attempts to evict the entry. A removal due to size may be ignored if the entry was updated and is no longer eligible for eviction.
func (c *BoundedLocalCache) evictEntry(node *Node) {
	if node == nil || len(node.Key) == 0 || !c.EnableEvict() {
		return
	}
	c.data.Remove(node.Key)
	c.weightedSize -= node.weight
	if node.inWindow() {
		c.windowWeightedSize -= node.weight
		c.windowDeque.Remove(node)
	} else if node.inMainProbation() {
		c.probationDeque.Remove(node)
	} else {
		c.mainProtectedWeightedSize -= node.weight
		c.protectedDeque.Remove(node)
	}
	return
}

// Evicts entries from the window space into the main space while the window size exceeds a maximum.
// return the number of candidate entries evicted from the window space
func (c *BoundedLocalCache) evictFromWindow() int {
	candidateNum := 0
	node := c.accessOrderWindowDeque().GetFront()
	for c.windowWeightedSize > c.windowMaximum {
		if node == nil {
			break
		}
		next := node.next
		c.accessOrderWindowDeque().Remove(node)
		c.accessOrderProbationDeque().PushBack(node)
		candidateNum++
		c.windowWeightedSize = c.windowWeightedSize - node.weight
		node = next
	}
	return candidateNum
}

// Evicts entries from the main space if the cache exceeds the maximum capacity. The main space
// determines whether admitting an entry (coming from the window space) is preferable to retaining
// the eviction policy's victim. This is decision is made using a frequency filter so that the
// least frequently used entry is removed.
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
		// Stop trying to evict candidates from window deque and always prefer the victim
		if candidates == 0 {
			candidate = nil
		}
		// Try evicting from the protected and window queues
		if candidate == nil && victim == nil {
			// Probation deque is empty now, try to evict from Protected deque
			if victimQueue == Probation {
				victim = c.accessOrderProtectedDeque().GetFront()
				victimQueue = Protected
				continue
			} else if victimQueue == Protected {
				// Both Probation deque and Protected deque are empty, try to evict from Window deque
				victim = c.accessOrderWindowDeque().GetFront()
				victimQueue = Window
				continue
			}
			// The pending operations will adjust the size to reflect the correct weight
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
			// candidate is nil, always prefer to evict victim from Probation or Protected
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
// return if the candidate should be admitted and the victim rejected
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

// onAccess updates the node's location in the page replacement policy
func (c *BoundedLocalCache) onAccess(n *Node) {
	if n == nil {
		return
	}
	if !c.EnableEvict() {
		return
	}
	key := n.Key
	if len(key) == 0 {
		return
	}
	c.sketch.increment(key)

	// update location
	if n.inWindow() && c.windowDeque.Contains(n) {
		c.windowDeque.MoveToBack(n)
	} else if n.inMainProbation() && c.probationDeque.Contains(n) {
		c.mainProtectedWeightedSize += n.weight
		c.probationDeque.Remove(n)
		c.protectedDeque.PushBack(n)
		n.makeIn(Protected)
	} else if n.inMainProtected() && c.protectedDeque.Contains(n) {
		c.protectedDeque.MoveToBack(n)
	}
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
	// Might lose some read record if readBuffer.offer return failed
	delayable := c.readBuffer.offer(unsafe.Pointer(node)) != full
	if c.shouldDrainBuffers(delayable) {
		c.scheduleDrainBuffers()
	}
}

func (c *BoundedLocalCache) afterWrite(t task) {
	for i := 0; i < WriteBufferRetries; i++ {
		if c.writeBuffer.offer(unsafe.Pointer(&t)) == success {
			c.scheduleAfterWrite()
			return
		}
		c.scheduleDrainBuffers()
	}

	// perform task directly
	c.performCleanUp(t)
}

func (c *BoundedLocalCache) shouldDrainBuffers(delayable bool) bool {
	switch c.drainState.get() {
	case Idle:
		return !delayable
	case Required:
		return true
	case ProcessingToIdle:
		return false
	case ProcessingToRequired:
		return false
	default:
		panic("illegal drain status.")
	}
}

func (c *BoundedLocalCache) scheduleAfterWrite() {
	status := c.drainState
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
			panic("illegal drain status.")
		}
	}
}

func (c *BoundedLocalCache) scheduleDrainBuffers() {
	status := c.drainState.get()
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
		if c.drainState.get() == Required {
			c.scheduleDrainBuffers()
		}
	}
}
