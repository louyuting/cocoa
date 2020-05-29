package cocoa

import "sync"

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
	return int(h ^ h>>32)
}

func (m *SegmentHashMap) getSegment(hash int) *Segment {
	return m.table[hash&m.mask]
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
