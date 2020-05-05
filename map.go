package cocoa

import "sync"

type ConcurrentMap interface {
	Put(key []byte, value interface{})

	PutIfAbsent(key []byte, value interface{}) (prior interface{})

	Get(key []byte) (value interface{}, existed bool)

	Remove(key []byte) (existed bool)

	Contains(key []byte) (ok bool)

	Len() int
}

const SegmentCount = 64

type SegmentHashMap struct {
	// length must be the power of 2
	table []*Segment
	// mast must be 2^n -1, for example: 0x00000000000000ff
	mask int
}

func newSegmentHashMap(size int) *SegmentHashMap {
	m := &SegmentHashMap{
		table: make([]*Segment, SegmentCount, SegmentCount),
		mask:  SegmentCount - 1,
	}
	for i := 0; i < SegmentCount; i++ {
		m.table[i] = &Segment{
			data: make(map[string]interface{}),
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

func (m *SegmentHashMap) Put(key []byte, value interface{}) {
	if len(key) == 0 {
		return
	}
	m.getSegment(m.hash(key)).Put(key, value)
}

func (m *SegmentHashMap) PutIfAbsent(key []byte, value interface{}) (prior interface{}) {
	if len(key) == 0 {
		return
	}
	return m.getSegment(m.hash(key)).PutIfAbsent(key, value)
}

func (m *SegmentHashMap) Get(key []byte) (value interface{}, existed bool) {
	if len(key) == 0 {
		return nil, false
	}
	return m.getSegment(m.hash(key)).Get(key)
}

func (m *SegmentHashMap) Remove(key []byte) (existed bool) {
	if len(key) == 0 {
		return false
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
	data map[string]interface{}
	mux  sync.RWMutex
}

func (s *Segment) Put(key []byte, value interface{}) {
	if key == nil || value == nil {
		return
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	s.data[*bytesToString(key)] = value
}

func (s *Segment) PutIfAbsent(key []byte, value interface{}) (prior interface{}) {
	if key == nil || value == nil {
		return
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

func (s *Segment) Get(key []byte) (value interface{}, existed bool) {
	if key == nil {
		return nil, false
	}
	s.mux.RLock()
	defer s.mux.RUnlock()
	value, existed = s.data[*bytesToString(key)]
	return
}

func (s *Segment) Remove(key []byte) (existed bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	_, existed = s.data[*bytesToString(key)]
	if existed {
		delete(s.data, *bytesToString(key))
	}
	return
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
