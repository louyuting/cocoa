package cocoa

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAccessOrderDeque(t *testing.T) {
	t.Run("TestAccessOrderDeque", func(t *testing.T) {
		q := AccessOrderDeque{}
		n1 := &Node{Value: 1}
		n2 := &Node{Value: 2}
		n3 := &Node{Value: 3}
		n4 := &Node{Value: 4}
		n5 := &Node{Value: 5}
		n6 := &Node{Value: 6}
		q.PushFront(n3)
		q.PushBack(n4)
		q.PushFront(n2)
		q.PushFront(n1)
		q.PushBack(n5)
		assert.True(t, q.Size() == 5)
		assert.True(t, q.RemoveFront() == n1)
		assert.True(t, q.GetFront() == n2)
		assert.True(t, q.GetBack() == n5)
		assert.True(t, q.Contains(n5))
		assert.True(t, !q.Contains(n6))
		assert.True(t, q.Size() == 4)
		assert.True(t, q.RemoveFront() == n2)
		assert.True(t, q.RemoveFront() == n3)
		assert.True(t, q.RemoveFront() == n4)
		assert.True(t, !q.IsEmpty())
		assert.True(t, q.RemoveFront() == n5)

		q.PushFront(n3)
		q.Clear()
		assert.True(t, q.Size() == 0)
	})
}
