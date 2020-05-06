package cocoa

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"unsafe"
)

type Test struct {
	name string
	age  int
}

func Test_atomicArray(t *testing.T) {
	t.Run("Test_atomicArray", func(t *testing.T) {
		a := newAtomicArray(100)
		t0 := &Test{
			name: "louyuting0",
			age:  19,
		}
		t1 := &Test{
			name: "louyuting1",
			age:  18,
		}
		t2 := &Test{
			name: "louyuting2",
			age:  19,
		}
		t3 := &Test{
			name: "louyuting3",
			age:  19,
		}
		t99 := &Test{
			name: "louyuting99",
			age:  19,
		}
		a.set(0, unsafe.Pointer(t0))
		a.set(1, unsafe.Pointer(t1))
		a.set(2, unsafe.Pointer(t2))
		a.set(3, unsafe.Pointer(t3))
		a.set(99, unsafe.Pointer(t99))

		t99Ptr := (*Test)(a.get(99))
		assert.True(t, a.len() == 100)
		assert.Equal(t, t99, t99Ptr)
	})
}
