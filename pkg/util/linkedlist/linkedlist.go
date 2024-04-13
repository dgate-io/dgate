package linkedlist

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrNodeNotFound  = errors.New("node not found")
	ErrValueNotFound = errors.New("value not found")
)

type LinkedList[T any] struct {
	Head *Node[T]
	Tail *Node[T]
	len  int
}

// New returns a new linked list.
func New[T any](items ...T) *LinkedList[T] {
	ll := &LinkedList[T]{}
	for _, item := range items {
		ll.Insert(item)
	}
	return ll
}

// Insert inserts a new node at the end of the linked list.
func (l *LinkedList[T]) Insert(value T) {
	node := &Node[T]{Value: value}
	l.len++
	if l.Head == nil {
		l.Head = node
		l.Tail = node
		return
	}
	l.Tail.Next = node
	l.Tail = node
}

// String returns a string representation of the linked list.
func (l *LinkedList[T]) String() string {
	var buf bytes.Buffer
	node := l.Head
	for node != nil {
		buf.WriteString(fmt.Sprintf("%v", node.Value))
		if node.Next != nil {
			buf.WriteString(" -> ")
		}
		node = node.Next
	}
	return buf.String()
}

// InsertBefore inserts a new node with the specififed value, before the given node.
func (l *LinkedList[T]) InsertBefore(node *Node[T], value T) error {
	newNode := &Node[T]{Value: value}
	if node == l.Head {
		newNode.Next = l.Head
		l.Head = newNode
		l.len++
		return nil
	}
	prev := l.Head
	for prev.Next != nil {
		if prev.Next == node {
			newNode.Next = prev.Next
			prev.Next = newNode
			l.len++
			return nil
		}
		prev = prev.Next
	}
	return ErrNodeNotFound
}

// Remove removes the given node from the linked list.
func (l *LinkedList[T]) Remove(value T) error {
	if l.Head == nil {
		return ErrValueNotFound
	}
	if reflect.DeepEqual(l.Head.Value, value) {
		if l.len == 1 {
			l.Head = nil
			l.Tail = nil
			l.len = 0
			return nil
		}
		l.Head = l.Head.Next
		l.len--
		return nil
	}
	prev := l.Head
	for prev.Next != nil {
		if reflect.DeepEqual(prev.Next.Value, value) {
			if prev.Next == l.Tail {
				l.Tail = prev
			}
			prev.Next = prev.Next.Next
			l.len--
			return nil
		}
		prev = prev.Next
	}
	return ErrValueNotFound
}

// RemoveTail removes the last node from the linked list.
func (l *LinkedList[T]) RemoveTail() error {
	if l.Head == nil {
		return ErrValueNotFound
	}
	if l.len == 1 {
		l.Head = nil
		l.Tail = nil
		l.len = 0
		return nil
	}
	prev := l.Head
	for prev.Next != nil {
		if prev.Next == l.Tail {
			prev.Next = nil
			l.Tail = prev
			l.len--
			return nil
		}
		prev = prev.Next
	}
	return ErrValueNotFound
}

// Contains returns true if the linked list contains the given value.
func (l *LinkedList[T]) Contains(value T) bool {
	if l.Head == nil {
		return false
	}
	node := l.Head
	for node != nil {
		if reflect.DeepEqual(node.Value, value) {
			return true
		}
		node = node.Next
	}
	return false
}

// Empty returns true if the linked list is empty.
func (l *LinkedList[T]) Empty() bool {
	return l.Head == nil
}

// Len returns the length of the linked list.
func (l *LinkedList[T]) Len() int {
	return l.len
}

// Node represents a node in the linked list.
type Node[T any] struct {
	Value T
	Next  *Node[T]
}
