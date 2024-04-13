package safe

import (
	"encoding/json"
	"sync/atomic"
)

// Ref is a generic struct that holds concurrent read and write objects
// to allow for safe concurrent read access with minimal locking, and
// atomic updates to the write object.
type Ref[T any] struct {
	v *atomic.Pointer[T]
}

// NewRef creates a new Ref instance with an initial value that is a pointer.
func NewRef[T any](t *T) *Ref[T] {
	ap := new(atomic.Pointer[T])
	if t != nil {
		ap.Store(shallowCopy(t))
	}
	return &Ref[T]{
		v: ap,
	}
}

// Read returns a copy of the object.
func (rw *Ref[T]) Read() *T {
	return shallowCopy(rw.v.Load())
}

// Load returns the current object.
func (rw *Ref[T]) Load() *T {
	return rw.v.Load()
}

// Replace replaces the ref and returns the old ref
func (rw *Ref[T]) Replace(new *T) (old *T) {
	old = rw.v.Load()
	newCopy := shallowCopy(new)
	rw.v.Store(newCopy)
	return old
}

// shallowCopy copies the contents of src to dst.
func shallowCopy[T any](src *T) *T {
	dst := new(T)
	*dst = *src
	return dst
}

// MarshalJSON returns the JSON encoding of the read object.
func (rw *Ref[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(rw.Read())
}

// UnmarshalJSON decodes the JSON data and updates the write object.
func (rw *Ref[T]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, rw.Read())
}
