package linkedlist

import (
	"fmt"
)

type Compariable interface {
	Compare(interface{}) int
}

func SortLinkedList[T any](ll *LinkedList[T], less func(i, j T) bool) {
	ll.Head = mergeSortListNode(ll.Head, less)
	ll.Tail = ll.Head
	for ll.Tail != nil && ll.Tail.Next != nil {
		ll.Tail = ll.Tail.Next
	}
}

// MergeSortList sorts a linked list using merge sort
func mergeSortListNode[T any](head *Node[T], less func(T, T) bool) *Node[T] {
	if head == nil || head.Next == nil {
		return head
	}

	// Find the middle of the list
	middle := findMiddle(head)

	// Split the list into two halves
	secondHalf := middle.Next
	middle.Next = nil

	// Recursively sort each half
	left := mergeSortListNode(head, less)
	right := mergeSortListNode(secondHalf, less)

	// Merge the sorted halves
	return merge(less, left, right)
}

// findMiddle finds the middle of the linked list
func findMiddle[T any](head *Node[T]) *Node[T] {
	slow, fast := head, head

	for fast != nil && fast.Next != nil && fast.Next.Next != nil {
		slow = slow.Next
		fast = fast.Next.Next
	}

	return slow
}

// merge merges two sorted linked lists
func merge[T any](less func(T, T) bool, left, right *Node[T]) *Node[T] {
	dummy := &Node[T]{}
	current := dummy

	for left != nil && right != nil {
		if less(left.Value, right.Value) {
			current.Next = left
			left = left.Next
		} else {
			current.Next = right
			right = right.Next
		}
		current = current.Next
	}

	// If there are remaining nodes in either list, append them
	if left != nil {
		current.Next = left
	}
	if right != nil {
		current.Next = right
	}

	return dummy.Next
}

// DisplayList prints the elements of the linked list
func DisplayList[T any](head *Node[T]) {
	current := head
	for current != nil {
		fmt.Printf("%v -> ", current.Value)
		current = current.Next
	}
	fmt.Println("nil")
}

func SortLinkedListIterative[T any](ll *LinkedList[T], less func(i, j T) bool) {
	ll.Head, ll.Tail = sortLinkedListNodeIterative(ll.Head, ll.Tail, less)
}

// MergeSortListIterative sorts a linked list using iterative merge sort
func sortLinkedListNodeIterative[T any](head, tail *Node[T], less func(i, j T) bool) (*Node[T], *Node[T]) {
	if head == nil || head.Next == nil {
		return head, tail
	}

	// Get the length of the linked list
	length := 0
	current := head
	for current != nil {
		length++
		current = current.Next
	}

	dummy := &Node[T]{}
	dummy.Next = head

	tailMaybe := tail

	for step := 1; step < length; step *= 2 {
		prevTail := dummy
		current = dummy.Next

		for current != nil {
			left := current
			right := split(left, step)
			current = split(right, step)

			merged := merge(less, left, right)
			prevTail.Next = merged

			// Move to the end of the merged list
			for merged.Next != nil {
				merged = merged.Next
			}

			prevTail = merged
			tailMaybe = merged
		}
	}
	for tailMaybe != nil && tailMaybe.Next != nil {
		tailMaybe = tailMaybe.Next
	}
	return dummy.Next, tailMaybe
}

// split splits the linked list into two parts, returns the head of the second part
func split[T any](head *Node[T], steps int) *Node[T] {
	for i := 1; head != nil && i < steps; i++ {
		head = head.Next
	}

	if head == nil {
		return nil
	}

	nextPart := head.Next
	head.Next = nil
	return nextPart
}
