package cocoa

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Ptr_Size(t *testing.T) {
	assert.True(t, PtrSize == 8)
}

func Test_ceilingPowerOfTwo(t *testing.T) {
	t.Run("Test_ceilingPowerOfTwo", func(t *testing.T) {
		assert.True(t, ceilingPowerOfTwo(0) == 1)
		assert.True(t, ceilingPowerOfTwo(2) == 2)
		assert.True(t, ceilingPowerOfTwo(7) == 8)
		assert.True(t, ceilingPowerOfTwo(44) == 64)
		assert.True(t, ceilingPowerOfTwo(111) == 128)
		assert.True(t, ceilingPowerOfTwo(222) == 256)
		assert.True(t, ceilingPowerOfTwo(1000) == 1024)
		assert.True(t, ceilingPowerOfTwo(-222) == 1)
		assert.True(t, ceilingPowerOfTwo(-1) == 1)
		assert.True(t, ceilingPowerOfTwo(256) == 256)
		assert.True(t, ceilingPowerOfTwo(1048576) == 1048576)
		assert.True(t, ceilingPowerOfTwo(1048575) == 1048576)
	})
}

func Test_bitCount(t *testing.T) {
	t.Run("Test_bitCount_1", func(t *testing.T) {
		assert.True(t, bitCount(OneMask) == 16)
	})

	t.Run("Test_bitCount_2", func(t *testing.T) {
		assert.True(t, bitCount(0xffff00ff) == 24)
	})

	t.Run("Test_bitCount_2", func(t *testing.T) {
		assert.True(t, bitCount(0xab) == 5)
	})
}
