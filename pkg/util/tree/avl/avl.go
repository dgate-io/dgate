package avl

import (
	"cmp"
	"sync"
)

// node represents a node in the AVL tree.
type node[K cmp.Ordered, V any] struct {
	key    K
	val    V
	left   *node[K, V]
	right  *node[K, V]
	height int
}

// Length returns the length of the node.
func (n *node[K, V]) _length() int {
	if n == nil {
		return 0
	}
	return 1 + n.left._length() + n.right._length()
}

// Tree represents an AVL tree.
type Tree[K cmp.Ordered, V any] interface {
	// Each traverses the tree in order and calls the given function for each node.
	Each(func(K, V) bool)
	// Insert inserts a key-value pair and returns the previous value if it exists.
	Insert(K, V) V
	// Delete removes a node with the given key from the AVL tree.
	Delete(K) bool
	// Pop removes a node with the given key from the AVL tree and returns the value.
	Pop(K) (V, bool)
	// Find returns the value associated with the given key.
	Find(K) (V, bool)
	// RootKeyValue returns the key and value of the root node.
	RootKeyValue() (K, V, bool)
	// Length returns the length of the tree.
	Length() int
	// Height returns the height of the tree.
	Height() int
	// Clear removes all nodes from the tree.
	Clear()
	// Empty returns true if the tree is empty.
	Empty() bool
	// Clone returns a copy of the tree.
	Clone() Tree[K, V]
}

type tree[K cmp.Ordered, V any] struct {
	root *node[K, V]
	mtx  *sync.RWMutex
}

func NewTree[K cmp.Ordered, V any]() Tree[K, V] {
	return &tree[K, V]{mtx: &sync.RWMutex{}}
}

func NewTreeFromRight[K cmp.Ordered, V any](t Tree[K, V]) Tree[K, V] {
	root := t.(*tree[K, V]).root
	if root != nil {
		root = root.right
	}
	return &tree[K, V]{root: root, mtx: &sync.RWMutex{}}
}

func NewTreeFromLeft[K cmp.Ordered, V any](t Tree[K, V]) Tree[K, V] {
	root := t.(*tree[K, V]).root
	if root != nil {
		root = root.left
	}
	return &tree[K, V]{root: root, mtx: &sync.RWMutex{}}
}

// Each traverses the tree in order and calls the given function for each node.
func (t *tree[K, V]) Each(f func(K, V) bool) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	each(t.root, func(n *node[K, V]) bool {
		return f(n.key, n.val)
	})
}

func each[K cmp.Ordered, V any](node *node[K, V], f func(*node[K, V]) bool) bool {
	if node == nil {
		return true
	}
	if each(node.left, f) && f(node) {
		return each(node.right, f)
	}
	return false
}

// rotateRight performs a right rotation on the given node.
func rotateRight[K cmp.Ordered, V any](y *node[K, V]) *node[K, V] {
	x := y.left
	t := x.right

	// Perform rotation
	x.right = y
	y.left = t

	// Update heights
	updateHeight(y)
	updateHeight(x)

	return x
}

// rotateLeft performs a left rotation on the given node.
func rotateLeft[K cmp.Ordered, V any](x *node[K, V]) *node[K, V] {
	y := x.right
	t := y.left

	// Perform rotation
	y.left = x
	x.right = t

	// Update heights
	updateHeight(x)
	updateHeight(y)

	return y
}

func getHeight[K cmp.Ordered, V any](node *node[K, V]) int {
	if node == nil {
		return 0
	}
	return node.height
}

func updateHeight[K cmp.Ordered, V any](node *node[K, V]) {
	node.height = 1 + max(getHeight(node.left), getHeight(node.right))
}

// getBalanceFactor returns the balance factor of a node.
func getBalanceFactor[K cmp.Ordered, V any](node *node[K, V]) int {
	if node == nil {
		return 0
	}
	if node.left == nil && node.right == nil {
		return 1
	}
	if node.left == nil {
		return -node.right.height
	}
	if node.right == nil {
		return node.left.height
	}
	return node.left.height - node.right.height
}

// Insert inserts a key-value pair into the AVL tree. thread-safe
func (t *tree[K, V]) Insert(key K, value V) (v V) {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	t.root, v = replace(t.root, key, value)
	return v
}

func replace[K cmp.Ordered, V any](root *node[K, V], key K, value V) (*node[K, V], V) {
	// Perform standard BST insertion
	var oldValue V
	if root == nil {
		return &node[K, V]{
			key:    key,
			val:    value,
			height: 1,
		}, oldValue
	}
	if key < root.key {
		root.left, oldValue = replace(root.left, key, value)
	} else if key > root.key {
		root.right, oldValue = replace(root.right, key, value)
	} else {
		// Update the value for an existing key
		oldValue, root.val = root.val, value
		return root, oldValue
	}

	updateHeight(root)

	// Get the balance factor of this node
	balance := getBalanceFactor(root)

	// Perform rotations if needed
	// Left Left Case
	if balance > 1 && key < root.left.key {
		return rotateRight(root), oldValue
	}
	// Right Right Case
	if balance < -1 && key > root.right.key {
		return rotateLeft(root), oldValue
	}
	// Left Right Case
	if balance > 1 && key > root.left.key {
		root.left = rotateLeft(root.left)
		return rotateRight(root), oldValue
	}
	// Right Left Case
	if balance < -1 && key < root.right.key {
		root.right = rotateRight(root.right)
		return rotateLeft(root), oldValue
	}

	return root, oldValue
}
func deleteNode[K cmp.Ordered, V any](root *node[K, V], key K) (*node[K, V], V, bool) {
	var v V
	if root == nil {
		return nil, v, false
	}

	deleted := false
	// Standard BST delete
	if key < root.key {
		root.left, v, deleted = deleteNode(root.left, key)
	} else if key > root.key {
		root.right, v, deleted = deleteNode(root.right, key)
	} else {
		deleted = true
		v = root.val
		// Node with only one child or no child
		if root.left == nil || root.right == nil {
			var temp *node[K, V]
			if root.left != nil {
				temp = root.left
			} else {
				temp = root.right
			}

			// No child case
			if temp == nil {
				// temp = root
				root = nil
			} else { // One child case
				*root = *temp // Copy the contents of the non-empty child
			}

			// Free the old node
			temp = nil
		} else {
			// Node with two children, get the inorder successor (smallest
			// in the right subtree)
			temp := findMin(root.right)

			// Copy the inorder successor's data to this node
			root.key = temp.key
			root.val = temp.val

			// Delete the inorder successor
			root.right, v, deleted = deleteNode(root.right, temp.key)
		}
	}

	// If the tree had only one node, then return
	if root == nil {
		return nil, v, deleted
	}

	if deleted {
		// Update height of the current node
		updateHeight(root)

		// Get the balance factor of this node
		balance := getBalanceFactor(root)

		// Perform rotations if needed
		// Left Left Case
		if balance > 1 && getBalanceFactor(root.left) >= 0 {
			return rotateRight(root), v, deleted
		}
		// Left Right Case
		if balance > 1 && getBalanceFactor(root.left) < 0 {
			root.left = rotateLeft(root.left)
			return rotateRight(root), v, deleted
		}
		// Right Right Case
		if balance < -1 && getBalanceFactor(root.right) <= 0 {
			return rotateLeft(root), v, deleted
		}
		// Right Left Case
		if balance < -1 && getBalanceFactor(root.right) > 0 {
			root.right = rotateRight(root.right)
			return rotateLeft(root), v, deleted
		}
	}

	return root, v, deleted
}

func findMin[K cmp.Ordered, V any](node *node[K, V]) *node[K, V] {
	for node.left != nil {
		node = node.left
	}
	return node
}

// Delete removes a node with the given key from the AVL tree.
func (t *tree[K, V]) Delete(key K) bool {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	var found bool
	t.root, _, found = deleteNode(t.root, key)
	return found
}

// Pop removes a node with the given key from the AVL tree and returns the value.
func (t *tree[K, V]) Pop(key K) (V, bool) {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	var found bool
	var v V
	t.root, v, found = deleteNode(t.root, key)
	return v, found
}

// Find returns the value associated with the given key.
func (t *tree[K, V]) Find(key K) (V, bool) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return find(t.root, key)
}

func find[K cmp.Ordered, V any](root *node[K, V], key K) (V, bool) {
	if root == nil {
		var v V
		return v, false
	}
	if key < root.key {
		return find(root.left, key)
	} else if key > root.key {
		return find(root.right, key)
	} else {
		return root.val, true
	}
}

// RootKey returns the key and value of the root node.
func (t *tree[K, V]) RootKeyValue() (K, V, bool) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	if t.root == nil {
		var k K
		var v V
		return k, v, false
	}
	return t.root.key, t.root.val, true
}

// Length returns the length of the tree.
func (t *tree[K, V]) Length() int {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	if t.root == nil {
		return 0
	}
	return t.root._length()
}

// Height returns the height of the tree.
func (t *tree[K, V]) Height() int {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	if t.root == nil {
		return 0
	}
	return t.root.height
}

// Clear removes all nodes from the tree.
func (t *tree[K, V]) Clear() {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	t.root = nil
}

// Clone returns a copy of the tree.
func (t *tree[K, V]) Clone() Tree[K, V] {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return &tree[K, V]{root: clone(t.root, func(_ K, v V) V {
		v2 := &v
		return *v2
	}), mtx: &sync.RWMutex{}}
}

// Empty returns true if the tree is empty.
func (t *tree[K, V]) Empty() bool {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return t.root == nil
}

func clone[K cmp.Ordered, V any](root *node[K, V], fn func(K, V) V) *node[K, V] {
	if root == nil {
		return nil
	}
	return &node[K, V]{
		key:   root.key,
		val:   fn(root.key, root.val),
		left:  clone(root.left, fn),
		right: clone(root.right, fn),
	}
}

// MarshalJSON returns the JSON encoding of the AVL tree.
// func (t *tree[K, V]) MarshalJSON() ([]byte, error) {
// 	t.mtx.RLock()
// 	defer t.mtx.RUnlock()
// 	return json.Marshal(t.root)
// }

// UnmarshalJSON decodes the JSON encoding of the AVL tree.
// func (t *tree[K, V]) UnmarshalJSON(data []byte) error {
// 	t.mtx.Lock()
// 	defer t.mtx.Unlock()
// 	return json.Unmarshal(data, &t.root)
// }
