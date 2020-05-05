package cocoa

// The async task for:
// 1. AddTask
// 2. UpdateTask
// 3. DeleteTask
// Add/Update/Delete the node to the page replacement policy.
// This task will not change the cache entity, and only to update the page replacement policy.
type task interface {
	run()
}

// AddTask is a async task used for add new kv to LocalCache
// Support size-base eviction
type AddTask struct {
	c      *BoundedLocalCache
	node   *Node
	weight int
}

func (a *AddTask) run() {
	c := a.c
	if !c.EnableEvict() {
		return
	}
	weightedSize := c.weightedSize
	c.weightedSize = weightedSize + a.weight

	node := a.node
	if len(node.Key) != 0 {
		c.sketch.increment(node.Key)
	}
	// insert to tail
	c.accessOrderWindowDeque().PushBack(node)
}

type UpdateTask struct {
	c          *BoundedLocalCache
	node       *Node
	weightDiff int
}

func (t *UpdateTask) run() {
	c := t.c
	if !c.EnableEvict() {
		return
	}
	node := t.node
	if node.inWindow() {
		c.windowWeightedSize = c.windowWeightedSize + t.weightDiff
	} else if node.inMainProtected() {
		c.mainProtectedWeightedSize = c.mainProtectedWeightedSize + t.weightDiff
	}
	//nodePtr := unsafe.Pointer(￿node)
	c.onAccess(nil)
}

type DeleteTask struct {
	c    *BoundedLocalCache
	node *Node
}

func (a *DeleteTask) run() {
	c := a.c
	if !c.EnableEvict() {
		return
	}
	node := a.node
	if node.inWindow() {
		c.accessOrderWindowDeque().Remove(node)
	} else if node.inMainProbation() {
		c.accessOrderProbationDeque().Remove(node)
	} else {
		c.accessOrderProtectedDeque().Remove(node)
	}
}
