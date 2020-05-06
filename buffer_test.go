package cocoa

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"unsafe"
)

func testElemConsumer(p unsafe.Pointer) {
	tPtr := (*Test)(p)
	fmt.Printf("%+v \n", tPtr)
}

func Test_RingBuffer(t *testing.T) {
	t.Run("Test_RingBuffer", func(t *testing.T) {
		buf := newRingBuffer()
		for i := 0; i < 16; i++ {
			buf.offer(unsafe.Pointer(&Test{
				name: "louyuting" + strconv.Itoa(i),
				age:  i,
			}))
		}
		assert.True(t, buf.offer(unsafe.Pointer(&Test{name: "louyuting", age: 18})) == full)
		buf.drainTo(testElemConsumer)

		assert.True(t, buf.r == 256 && buf.w == 256)
	})
}
