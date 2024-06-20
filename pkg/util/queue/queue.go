package queue

type Queue[V any] interface {
	// Push adds an element to the queue.
	Push(V)
	// Pop removes and returns the element at the front of the queue.
	Pop() (V, bool)
	// Peek returns the element at the front of the queue without removing it.
	// It returns nil if the queue is empty.
	Peek() (V, bool)
	// Len returns the number of elements in the queue.
	Len() int
}

type queueImpl[V any] struct {
	items []V
}

// New returns a new queue.
func New[V any](vs ...V) Queue[V] {
	if len(vs) > 0 {
		q := newQueue[V](len(vs))
		for _, v := range vs {
			q.Push(v)
		}
		return q
	}
	return newQueue[V](128)
}

// NewWithSize returns a new queue with the specified size.
func NewWithSize[V any](size int) Queue[V] {
	return newQueue[V](size)
}

func newQueue[V any](size int) *queueImpl[V] {
	return &queueImpl[V]{
		items: make([]V, 0, size),
	}
}

func (q *queueImpl[V]) Push(item V) {
	q.items = append(q.items, item)
}

func (q *queueImpl[V]) Pop() (V, bool) {
	var item V
	if len(q.items) == 0 {
		return item, false
	}
	item = q.items[0]
	q.items = q.items[1:]
	return item, true
}

func (q *queueImpl[V]) Peek() (V, bool) {
	if len(q.items) == 0 {
		var v V
		return v, false
	}
	return q.items[0], true
}

func (q *queueImpl[V]) Len() int {
	return len(q.items)
}
