package cocoa

type QueueType int32

const (
	Window QueueType = iota
	Probation
	Protected
)

type Node struct {
	Key        []byte
	Value      interface{}
	weight     int
	prev, next *Node
	dequeIn    QueueType
}

func (n *Node) makeIn(newQueueType QueueType) {
	n.dequeIn = newQueueType
}

func (n *Node) inWindow() bool {
	return n.dequeIn == Window
}

func (n *Node) inMainProbation() bool {
	return n.dequeIn == Probation
}

func (n *Node) inMainProtected() bool {
	return n.dequeIn == Protected
}
