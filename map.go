package cocoa

import "sync"

//type computeValue func(key []byte) *Node
//
//type ConcurrentMap interface {
//	Put(key []byte, value *Node)
//
//	PutIfAbsent(key []byte, value *Node) (prior *Node)
//
//	// ComputeIfAbsent attempts to compute its value using the given mapping function and enters it into this map
//	// If the specified key is not already associated with a value.
//	// return the current (existing or computed) value associated with the specified key, or null if the computed value is null
//	ComputeIfAbsent(key []byte, compute computeValue) *Node
//
//	Get(key []byte) (value *Node, existed bool)
//
//	Remove(key []byte) (existed bool)
//
//	Contains(key []byte) (ok bool)
//
//	Len() int
//}

const SegmentCount = 64

type SegmentHashMap struct {
	// length must be the power of 2
	table []*Segment
	// mast must be 2^n -1, for example: 0x00000000000000ff
	mask int
}

func newSegmentHashMap() *SegmentHashMap {
	m := &SegmentHashMap{
		table: make([]*Segment, SegmentCount, SegmentCount),
		mask:  SegmentCount - 1,
	}
	for i := 0; i < SegmentCount; i++ {
		m.table[i] = &Segment{
			data: make(map[string]*Node),
			mux:  sync.RWMutex{},
		}
	}
	return m
}

func (m *SegmentHashMap) hash(key []byte) int {
	if len(key) == 0 {
		return 0
	}
	h := hash(key)
	return h ^ h>>32
}

func (m *SegmentHashMap) getSegment(hash int) *Segment {
	return m.table[hash&m.mask]
}

func (m *SegmentHashMap) Put(key []byte, value *Node) (prior *Node) {
	if len(key) == 0 {
		return nil
	}
	return m.getSegment(m.hash(key)).Put(key, value)
}

func (m *SegmentHashMap) PutIfAbsent(key []byte, value *Node) (prior *Node) {
	if len(key) == 0 {
		return
	}
	return m.getSegment(m.hash(key)).PutIfAbsent(key, value)
}

func (m *SegmentHashMap) Get(key []byte) (value *Node, existed bool) {
	if len(key) == 0 {
		return nil, false
	}
	return m.getSegment(m.hash(key)).Get(key)
}

func (m *SegmentHashMap) Remove(key []byte) (prior *Node) {
	if len(key) == 0 {
		return nil
	}
	return m.getSegment(m.hash(key)).Remove(key)
}

func (m *SegmentHashMap) Contains(key []byte) (ok bool) {
	if len(key) == 0 {
		return false
	}
	return m.getSegment(m.hash(key)).Contains(key)
}

func (m *SegmentHashMap) Len() int {
	count := 0
	for i := 0; i < len(m.table); i++ {
		count += m.table[i].Len()
	}
	return count
}

type Segment struct {
	data map[string]*Node
	mux  sync.RWMutex
}

func (s *Segment) PutOnlyUpdate(key []byte, newNode *Node) (prior *Node) {
	if key == nil || newNode == nil {
		panic("key or node is nil")
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	priorNode, existed := s.data[*bytesToString(key)]
	if existed {
		priorNode.Value = newNode.Value
		return priorNode
	} else {
		return nil
	}
}

func (s *Segment) PutIfAbsent(key []byte, value *Node) (prior *Node) {
	if key == nil || value == nil {
		panic("key or value is nil")
	}
	s.mux.RLock()
	prior, existed := s.data[*bytesToString(key)]
	s.mux.RUnlock()
	if existed {
		return prior
	}

	s.mux.Lock()
	defer s.mux.Unlock()
	prior, existed = s.data[*bytesToString(key)]
	if existed {
		return prior
	}
	s.data[*bytesToString(key)] = value
	return nil
}

func (s *Segment) Get(key []byte) (value *Node, existed bool) {
	if key == nil {
		panic("key is nil")
	}
	s.mux.RLock()
	defer s.mux.RUnlock()
	value, existed = s.data[*bytesToString(key)]
	return
}

func (s *Segment) Remove(key []byte) (prior *Node) {
	s.mux.Lock()
	defer s.mux.Unlock()
	priorNode, existed := s.data[*bytesToString(key)]
	if existed {
		delete(s.data, *bytesToString(key))
	}
	return priorNode
}

func (s *Segment) Contains(key []byte) (ok bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	_, ok = s.data[*bytesToString(key)]
	return
}

func (s *Segment) Len() int {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return len(s.data)
}
