package cocoa

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAccessOrderDeque(t *testing.T) {
	t.Run("", func(t *testing.T) {
		q := AccessOrderDeque{}
		n1 := &Node{Value: 1}
		n2 := &Node{Value: 2}
		n3 := &Node{Value: 3}
		n4 := &Node{Value: 4}
		n5 := &Node{Value: 5}
		q.PushFront(n3)
		q.PushBack(n4)
		q.PushFront(n2)
		q.PushFront(n1)
		q.PushBack(n5)
		assert.True(t, q.Size() == 5)
	})
}
