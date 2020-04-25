package cocoa

import (
	"fmt"
	"testing"
)

func Test_memhash(t *testing.T) {
	t.Run("Test_memhash", func(t *testing.T) {
		s := make([]byte, 0)
		fmt.Println(hash(s))
	})
}
