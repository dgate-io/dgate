package heap

import (
	"cmp"
	"errors"
)

type HeapType int

const (
	MinHeapType HeapType = iota
	MaxHeapType
)

type Heap[K cmp.Ordered, V any] struct {
	heapType HeapType
	data     []pair[K, V]
}

type pair[K cmp.Ordered, V any] struct {
	key K
	val V
}

var ErrHeapEmpty = errors.New("heap is empty")

func NewHeap[K cmp.Ordered, V any](ht HeapType) *Heap[K, V] {
	if ht != MinHeapType && ht != MaxHeapType {
		panic("invalid heap type")
	}
	return &Heap[K, V]{ht, []pair[K, V]{}}
}

func (h *Heap[K, V]) Push(key K, val V) {
	h.data = append(h.data, pair[K, V]{
		key: key,
		val: val,
	})
	h.heapifyUp(len(h.data)-1, h.heapType)
}

func (h *Heap[K, V]) Pop() (K, V, bool) {
	if len(h.data) == 0 {
		var (
			v V
			k K
		)
		return k, v, false
	}
	min := h.data[0]
	lastIdx := len(h.data) - 1
	h.data[0] = h.data[lastIdx]
	h.data = h.data[:lastIdx]
	h.heapifyDown(0, h.heapType)
	return min.key, min.val, true
}

func (h *Heap[K, V]) Peak() (K, V, bool) {
	if len(h.data) == 0 {
		var (
			v V
			k K
		)
		return k, v, false
	}
	min := h.data[0]
	return min.key, min.val, true
}

func (h *Heap[K, V]) Len() int {
	return len(h.data)
}

func (h *Heap[K, V]) heapifyDown(idx int, ht HeapType) {
	for {
		left := 2*idx + 1
		right := 2*idx + 2
		target := idx
		if ht == MinHeapType {
			if left < len(h.data) && h.data[left].key < h.data[target].key {
				target = left
			}
			if right < len(h.data) && h.data[right].key < h.data[target].key {
				target = right
			}
			if target == idx {
				break
			}
		} else {
			if left < len(h.data) && h.data[left].key > h.data[target].key {
				target = left
			}
			if right < len(h.data) && h.data[right].key > h.data[target].key {
				target = right
			}
			if target == idx {
				break
			}
		}
		h.data[idx], h.data[target] = h.data[target], h.data[idx]
		idx = target
	}
}

func (h *Heap[K, V]) heapifyUp(idx int, ht HeapType) {
	for idx > 0 {
		parent := (idx - 1) / 2
		if ht == MinHeapType {
			if h.data[parent].key <= h.data[idx].key {
				break
			}
		} else {
			if h.data[parent].key >= h.data[idx].key {
				break
			}
		}
		h.data[parent], h.data[idx] = h.data[idx], h.data[parent]
		idx = parent
	}
}
