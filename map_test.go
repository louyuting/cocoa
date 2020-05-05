package cocoa

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestSegmentHashMap(t *testing.T) {
	t.Run("newSegmentHashMap", func(t *testing.T) {
		m := newSegmentHashMap(1)
		assert.True(t, m.Len() == 0)
	})
}

func BenchmarkMap_Get(b *testing.B) {
	b.Run("BenchmarkSegment_Put", func(b *testing.B) {
		m := &Segment{
			data: make(map[string]interface{}),
			mux:  sync.RWMutex{},
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			m.Put(([]byte)(uuid.New().String()), 1)
		}
	})

	b.Run("BenchmarkSegmentHashMap_Put", func(b *testing.B) {
		m := newSegmentHashMap(1)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			m.Put(([]byte)(uuid.New().String()), 1)
		}
	})

	b.Run("BenchmarkSegment_Put_Concurrent_10", func(b *testing.B) {
		m := &Segment{
			data: make(map[string]interface{}),
			mux:  sync.RWMutex{},
		}
		b.ReportAllocs()
		b.ResetTimer()
		b.SetParallelism(10)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Put(([]byte)(uuid.New().String()), 1)
			}
		})
	})

	b.Run("BenchmarkSegmentHashMap_Concurrent_10", func(b *testing.B) {
		m := newSegmentHashMap(1)
		b.ResetTimer()
		b.ReportAllocs()
		b.SetParallelism(10)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Put(([]byte)(uuid.New().String()), 1)
			}
		})
	})

	b.Run("BenchmarkSegment_Put_Concurrent_100", func(b *testing.B) {
		m := &Segment{
			data: make(map[string]interface{}),
			mux:  sync.RWMutex{},
		}
		b.ReportAllocs()
		b.ResetTimer()
		b.SetParallelism(100)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Put(([]byte)(uuid.New().String()), 1)
			}
		})
	})

	b.Run("BenchmarkSegmentHashMap_Concurrent_100", func(b *testing.B) {
		m := newSegmentHashMap(1)
		b.ResetTimer()
		b.ReportAllocs()
		b.SetParallelism(100)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Put(([]byte)(uuid.New().String()), 1)
			}
		})
	})

	b.Run("BenchmarkSegment_Put_Concurrent_1000", func(b *testing.B) {
		m := &Segment{
			data: make(map[string]interface{}),
			mux:  sync.RWMutex{},
		}
		b.ReportAllocs()
		b.ResetTimer()
		b.SetParallelism(1000)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Put(([]byte)(uuid.New().String()), 1)
			}
		})
	})

	b.Run("BenchmarkSegmentHashMap_Concurrent_1000", func(b *testing.B) {
		m := newSegmentHashMap(1)
		b.ResetTimer()
		b.ReportAllocs()
		b.SetParallelism(1000)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Put(([]byte)(uuid.New().String()), 1)
			}
		})
	})
}
