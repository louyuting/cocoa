package cocoa

// task is the async task for:
// 1. AddTask
// 2. UpdateTask
// 3. DeleteTask
// Add/Update/Delete the node to the page replacement policy.
// This task will not change the cache entity, and only to update the page replacement policy.
type task interface {
	run()
}

type ReadTask struct {
	c    *BoundedLocalCache
	node *Node
}

func (t *ReadTask) run() {
	t.c.onAccess(t.node)
}

// AddTask is a async task used for add new kv to LocalCache
// Support size-base eviction
type AddTask struct {
	c      *BoundedLocalCache
	node   *Node
	weight int
}

// AddTask update node's the weight and frequency
func (t *AddTask) run() {
	c := t.c
	if !c.EnableEvict() {
		return
	}
	// update cache size
	c.weightedSize += t.weight
	// update window deque size
	c.windowWeightedSize += t.weight

	node := t.node
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

// UpdateTask update the node's frequency and location
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
	} else if node.inMainProbation() {
		// will move to protected deque
		// do nothing
	}
	c.onAccess(node)
}

type DeleteTask struct {
	c    *BoundedLocalCache
	node *Node
}

// DeleteTask remove the node from deque
func (t *DeleteTask) run() {
	c := t.c
	if !c.EnableEvict() {
		return
	}
	node := t.node
	if node.inWindow() {
		c.accessOrderWindowDeque().Remove(node)
		c.windowWeightedSize -= node.weight
	} else if node.inMainProbation() {
		c.accessOrderProbationDeque().Remove(node)
	} else {
		c.accessOrderProtectedDeque().Remove(node)
		c.mainProtectedWeightedSize -= node.weight
	}
}
