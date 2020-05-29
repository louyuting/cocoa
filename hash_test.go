package cocoa

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_memhash(t *testing.T) {
	t.Run("Test_memhash_empty_slice", func(t *testing.T) {
		s := make([]byte, 0)
		assert.True(t, hash(s) == 0)
	})

	t.Run("Test_memhash_distribution", func(t *testing.T) {
		var table [1000]int
		for i := 0; i < 10000000; i++ {
			h := hash([]byte(uuid.New().String()))
			idx := h % 1000
			table[idx] = table[idx] + 1
		}
		fmt.Printf("%+v\n", table)
		for idx, e := range table {
			fmt.Println(idx, ":", (float64(e)/float64(10000))*100, "%")
		}
	})
}
