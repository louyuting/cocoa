package cocoa

import (
	"fmt"
	"testing"
)

func TestFrequencySketch_increment(t *testing.T) {
	t.Run("TestFrequencySketch_increment", func(t *testing.T) {
		f := NewFrequencySketch(512)
		i := 0
		key := []byte{'l', 'o', 'u'}
		for i < 10 {
			f.increment(key)
			i++
		}
		fmt.Println(f.frequency(key))
	})
}
