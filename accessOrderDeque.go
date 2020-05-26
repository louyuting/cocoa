package cocoa

type OrderDeque interface {
	PushFront(n *Node)
	PushBack(n *Node)

	GetFront() *Node
	GetBack() *Node
	GetPrevious(n *Node) *Node
	GetNext(n *Node) *Node

	// return whether n is existed
	Remove(n *Node) bool
	// Return front node
	RemoveFront() *Node
	// Return back node
	RemoveBack() *Node

	MoveToFront(n *Node)
	MoveToBack(n *Node)

	// Removes all of the elements from this collection
	Clear()

	ToSlice() []*Node

	IsEmpty() bool
	Contains(n *Node) bool

	Size() int
}

type AccessOrderDeque struct {
	// the oldest node to remove next.
	head *Node
	// the most frequency node
	tail *Node
}

func (q *AccessOrderDeque) PushFront(n *Node) {
	if q.Contains(n) {
		return
	}
	h := q.head
	q.head = n
	if h == nil {
		// the deque is empty
		q.tail = n
	} else {
		h.prev = n
		n.next = h
	}
}

func (q *AccessOrderDeque) PushBack(n *Node) {
	if q.Contains(n) {
		return
	}
	t := q.tail
	q.tail = n
	if t == nil {
		q.head = n
	} else {
		t.next = n
		n.prev = t
	}
}

func (q *AccessOrderDeque) GetFront() *Node {
	if q.IsEmpty() {
		return nil
	}
	return q.head
}

func (q *AccessOrderDeque) GetBack() *Node {
	if q.IsEmpty() {
		return nil
	}
	return q.tail
}

func (q *AccessOrderDeque) GetPrevious(n *Node) *Node {
	return n.prev
}

func (q *AccessOrderDeque) GetNext(n *Node) *Node {
	return n.next
}

func (q *AccessOrderDeque) Remove(n *Node) bool {
	if !q.Contains(n) {
		return false
	}
	prev := n.prev
	next := n.next
	if prev == nil {
		q.head = n
	} else {
		prev.next = next
		n.prev = nil
	}

	if next == nil {
		q.tail = n
	} else {
		next.prev = prev
		n.next = nil
	}

	return true
}

func (q *AccessOrderDeque) RemoveFront() *Node {
	if q.IsEmpty() {
		return nil
	}
	h := q.head
	next := h.next
	h.next = nil
	q.head = next

	// only one node in deque
	if next == nil {
		q.tail = nil
	} else {
		// multi node in deque
		next.prev = nil
	}
	return h
}

func (q *AccessOrderDeque) RemoveBack() *Node {
	if q.IsEmpty() {
		return nil
	}
	t := q.tail
	prev := t.prev
	t.prev = nil
	q.tail = prev

	if prev == nil {
		q.head = nil
	} else {
		prev.next = nil
	}
	return t
}

func (q *AccessOrderDeque) MoveToFront(n *Node) {
	if !q.Contains(n) {
		return
	}
	if n == q.head {
		return
	}
	q.Remove(n)
	q.PushFront(n)
}

func (q *AccessOrderDeque) MoveToBack(n *Node) {
	if !q.Contains(n) {
		return
	}
	if n == q.tail {
		return
	}
	q.Remove(n)
	q.PushBack(n)
}

func (q *AccessOrderDeque) Clear() {
	for cur := q.head; cur != nil; {
		next := cur.next
		cur.prev = nil
		cur.next = nil
		cur = next
	}
	q.head = nil
	q.tail = nil
}

func (q *AccessOrderDeque) ToSlice() []*Node {
	size := q.Size()
	ret := make([]*Node, size, size)
	for cur := q.head; cur != nil; cur = cur.next {
		ret = append(ret, cur)
	}
	return ret
}

func (q *AccessOrderDeque) IsEmpty() bool {
	return q.head == nil
}

func (q *AccessOrderDeque) Contains(n *Node) bool {
	return n.prev != nil || n.next != nil || n == q.head
}

func (q *AccessOrderDeque) Size() int {
	n := 0
	for cur := q.head; cur != nil; cur = cur.next {
		n++
	}
	return n
}
