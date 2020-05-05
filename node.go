package cocoa

type QueueType int32

const (
	Window QueueType = iota
	Probation
	Protected
)

type Node struct {
	Key   []byte
	Value interface{}

	weight int

	prev, next *Node

	queueIn QueueType
}

func (n *Node) getWeight() int {
	return n.weight
}
func (n *Node) setWeight(weight int) {
	n.weight = weight
}

func (n *Node) getNextInAccessOrder() *Node {
	return n.next
}

func (n *Node) setNextInAccessOrder(next *Node) {
	n.next = next
}

func (n *Node) getPreviousInAccessOrder() *Node {
	return n.prev
}

func (n *Node) setPreviousInAccessOrder(prev *Node) {
	n.prev = prev
}

func (n *Node) inWindow() bool {
	return n.queueIn == Window
}

func (n *Node) inMainProbation() bool {
	return n.queueIn == Probation
}

func (n *Node) inMainProtected() bool {
	return n.queueIn == Protected
}
