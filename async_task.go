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

type AddTask struct {
	node   *Node
	weight int
}

func (a *AddTask) run() {
	panic("implement me")
}

type UpdateTask struct {
	node       *Node
	weightDiff int
}

func (a *UpdateTask) run() {
	panic("implement me")
}

type DeleteTask struct {
	node *Node
}

func (a *DeleteTask) run() {
	panic("implement me")
}
