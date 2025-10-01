package dockerserver

import "sync/atomic"

type AtomicBool struct {
	v atomic.Int32
}

// Set assigns the boolean atomically.
func (b *AtomicBool) Set(val bool) {
	if val {
		b.v.Store(1)
	} else {
		b.v.Store(0)
	}
}

// Get reads the boolean atomically.
func (b *AtomicBool) Get() bool {
	return b.v.Load() != 0
}

// CompareAndSwap swaps the value if the old matches.
func (b *AtomicBool) CompareAndSwap(old, new bool) bool {
	var o, n int32
	if old {
		o = 1
	}
	if new {
		n = 1
	}
	return b.v.CompareAndSwap(o, n)
}
