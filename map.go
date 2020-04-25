package cocoa

import "sync"

type ConcurrentMap interface {
	Put(key interface{}, value interface{})

	PutIfAbsent(key interface{}, value interface{}) (prior interface{})

	Get(key interface{}) (value interface{}, existed bool)

	Remove(key interface{}) (existed bool)

	Contains(key interface{}) (ok bool)

	Len() int
}

type SafeMap struct {
	data map[interface{}]interface{}
	mux  *sync.RWMutex
}

func (s *SafeMap) Put(key interface{}, value interface{}) {
	if key == nil || value == nil {
		return
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	s.data[key] = value
}

func (s *SafeMap) PutIfAbsent(key interface{}, value interface{}) (prior interface{}) {
	if key == nil || value == nil {
		return
	}
	s.mux.RLock()
	prior, existed := s.data[key]
	s.mux.RUnlock()
	if existed {
		return prior
	}

	s.mux.Lock()
	defer s.mux.Unlock()
	prior, existed = s.data[key]
	if existed {
		return prior
	}
	s.data[key] = value
	return nil
}

func (s *SafeMap) Get(key interface{}) (value interface{}, existed bool) {
	if key == nil {
		return nil, false
	}
	s.mux.RLock()
	defer s.mux.RUnlock()
	value, existed = s.data[key]
	return
}

func (s *SafeMap) Remove(key interface{}) (existed bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	_, existed = s.data[key]
	if existed {
		delete(s.data, key)
	}
	return
}

func (s *SafeMap) Contains(key interface{}) (ok bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	_, ok = s.data[key]
	return
}

func (s *SafeMap) Len() int {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return len(s.data)
}
