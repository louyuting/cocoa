package cocoa

import (
	"fmt"
	"testing"
)

func TestDrainStatus(t *testing.T) {
	t.Run("TestDrainStatus", func(t *testing.T) {
		s := Idle
		s.set(ProcessingToRequired)
		fmt.Println(s.get() >= ProcessingToIdle)
	})
}
