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
	queueIn    QueueType
}

func (n *Node) makeIn(newQueueType QueueType) {
	n.queueIn = newQueueType
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
