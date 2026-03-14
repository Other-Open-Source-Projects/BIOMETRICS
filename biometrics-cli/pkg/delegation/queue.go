package delegation

import (
	"container/heap"
	"sync"
)

type PriorityQueue struct {
	tasks []*Task
	mu    sync.RWMutex
}

func NewPriorityQueue() *PriorityQueue {
	return &PriorityQueue{
		tasks: make([]*Task, 0),
	}
}

func (pq *PriorityQueue) Len() int {
	return len(pq.tasks)
}

func (pq *PriorityQueue) Less(i, j int) bool {
	if pq.tasks[i].Priority == pq.tasks[j].Priority {
		return pq.tasks[i].CreatedAt.Before(pq.tasks[j].CreatedAt)
	}
	return pq.tasks[i].Priority < pq.tasks[j].Priority
}

func (pq *PriorityQueue) Swap(i, j int) {
	pq.tasks[i], pq.tasks[j] = pq.tasks[j], pq.tasks[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	task := x.(*Task)
	pq.tasks = append(pq.tasks, task)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := pq.tasks
	n := len(old)
	task := old[n-1]
	pq.tasks = old[0 : n-1]
	return task
}

func (pq *PriorityQueue) Enqueue(task *Task) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	heap.Push(pq, task)
}

func (pq *PriorityQueue) Dequeue() *Task {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if len(pq.tasks) == 0 {
		return nil
	}
	return heap.Pop(pq).(*Task)
}

func (pq *PriorityQueue) Peek() *Task {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	if len(pq.tasks) == 0 {
		return nil
	}
	return pq.tasks[0]
}

func (pq *PriorityQueue) Size() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return len(pq.tasks)
}

func (pq *PriorityQueue) IsEmpty() bool {
	return pq.Size() == 0
}
