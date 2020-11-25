package pool

import (
	"reflect"
	"sync"
	"sync/atomic"
)

// ReferenceCountable ...
type ReferenceCountable interface {
	SetInstance(i interface{})
	IncrementReferenceCount()
	DecrementReferenceCount()
}

// ReferenceCounter ...
type ReferenceCounter struct {
	count       *uint32
	destination *sync.Pool
	released    *uint32
	instance    interface{}
	reset       func(interface{}) error
	id          uint32
}

// IncrementReferenceCount Method to increment a reference
func (r ReferenceCounter) IncrementReferenceCount() {
	atomic.AddUint32(r.count, 1)
}

// DecrementReferenceCount Method to decrement a reference
// If the reference count goes to zero, the object is put back inside the pool
func (r ReferenceCounter) DecrementReferenceCount() {
	if atomic.LoadUint32(r.count) == 0 {
		panic("this should not happen =>" + reflect.TypeOf(r.instance).String())
	}
	if atomic.AddUint32(r.count, ^uint32(0)) == 0 {
		atomic.AddUint32(r.released, 1)
		if err := r.reset(r.instance); err != nil {
			panic("error while resetting an instance => " + err.Error())
		}
		r.destination.Put(r.instance)
		r.instance = nil
	}
}

// SetInstance Method to set the current instance
func (r *ReferenceCounter) SetInstance(i interface{}) {
	r.instance = i
}

// ReferenceCountedPool Struct representing the pool
type ReferenceCountedPool struct {
	pool       *sync.Pool
	factory    func() ReferenceCountable
	returned   uint32
	allocated  uint32
	referenced uint32
}

// NewReferenceCountedPool Method to create a new pool
func NewReferenceCountedPool(factory func(referenceCounter ReferenceCounter) ReferenceCountable, reset func(interface{}) error) *ReferenceCountedPool {
	p := new(ReferenceCountedPool)
	p.pool = new(sync.Pool)
	p.pool.New = func() interface{} {
		// Incrementing allocated count
		atomic.AddUint32(&p.allocated, 1)
		c := factory(ReferenceCounter{
			count:       new(uint32),
			destination: p.pool,
			released:    &p.returned,
			reset:       reset,
			id:          p.allocated,
		})
		return c
	}
	return p
}

// Get Method to get new object
func (p *ReferenceCountedPool) Get() ReferenceCountable {
	c := p.pool.Get().(ReferenceCountable)
	c.SetInstance(c)
	atomic.AddUint32(&p.referenced, 1)
	c.IncrementReferenceCount()
	return c
}

// Stats Method to return reference counted pool stats
func (p *ReferenceCountedPool) Stats() map[string]interface{} {
	return map[string]interface{}{"allocated": p.allocated, "referenced": p.referenced, "returned": p.returned}
}
