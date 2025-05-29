// PoolOf is a generic wrapper around sync.Pool for managing reusable objects of type V.
// It allows optional customization of object creation and reuse behavior via user-provided functions.
//
// Fields:
//   - reuseFn: Optional function called with the value when it is retrieved from the pool,
//     typically used to reset or prepare the value for reuse.
//
// NewPoolOf creates a new PoolOf for a specific type V.
//   - newFn: Function to create a new instance of V when the pool is empty (required).
//   - reuseFn: Optional function called with the value when it is retrieved from the pool.
//     If provided, it should reset or prepare the value for reuse.
//
// Get retrieves a value from the pool, applies reuseFn if provided, and returns the value.
//   - Panics if the type assertion fails (i.e., the stored value is not of type V).
package recorder

import (
	"sync"
)

type PoolOf[V any] struct {
	*sync.Pool
	reuseFn func(V) // Optional function called with the value when it is retrieved from the pool,
}

func NewPoolOf[V any](newFn func() V, reuseFn func(V)) *PoolOf[V] {
	if newFn == nil {
		newFn = func() V { return *new(V) }
	}
	return &PoolOf[V]{
		Pool: &sync.Pool{
			New: func() any {
				return newFn() // Create a new instance of V using the provided function
			},
		},
		reuseFn: reuseFn,
	}
}

func (p *PoolOf[V]) Get() (value V) {
	v, ok := p.Pool.Get().(V)
	if !ok {
		panic("type assertion failed, expected type does not match")
	}
	if p.reuseFn != nil {
		p.reuseFn(v)
	}
	return v
}
