package ringbuffer

import (
	"sync/atomic"
)

type RingBufferWithoutAlign[T any] struct {
	cap   uint32
	mask  uint32
	write uint32
	read  uint32
	data  []*T
}

func NewRingBufferWithoutAlign[T any](capacity int) *RingBufferWithoutAlign[T] {
	return &RingBufferWithoutAlign[T]{
		cap:   uint32(capacity),
		mask:  uint32(capacity - 1),
		write: 0,
		read:  0,
		data:  make([]*T, capacity),
	}
}

func (r *RingBufferWithoutAlign[T]) Write(value T) (ok bool) {
	oldWrite := atomic.LoadUint32(&r.write)
	oldRead := atomic.LoadUint32(&r.read)

	if r.isFull(oldWrite, oldRead) {
		return false
	}
	newWrite := oldWrite + 1
	elem := r.data[newWrite&r.mask]
	if elem != nil {
		return false
	}
	if !atomic.CompareAndSwapUint32(&r.write, oldWrite, newWrite) {
		return false
	}

	r.data[newWrite&r.mask] = &value
	return true
}

func (r *RingBufferWithoutAlign[T]) Read() (val T, ok bool) {
	oldWrite := atomic.LoadUint32(&r.write)
	oldRead := atomic.LoadUint32(&r.read)

	if r.isEmpty(oldWrite, oldRead) {
		return
	}
	newRead := oldRead + 1
	elem := r.data[newRead&r.mask]
	if elem == nil {
		return
	}

	if !atomic.CompareAndSwapUint32(&r.read, oldRead, newRead) {
		return
	}
	return *elem, true
}

func (r *RingBufferWithoutAlign[T]) ReadVec(buf []T) (read uint32) {
	oldWrite := atomic.LoadUint32(&r.write)
	oldRead := atomic.LoadUint32(&r.read)

	if r.isEmpty(oldWrite, oldRead) {
		return
	}

	cur := oldRead + 1
	for ; cur <= oldWrite; cur++ {
		elem := r.data[cur&r.mask]
		if elem == nil {
			break
		}
		buf[cur-oldRead-1] = *elem
		r.data[cur&r.mask] = nil
	}

	atomic.StoreUint32(&r.read, cur-1)

	return cur - oldRead - 1
}

func (r *RingBufferWithoutAlign[T]) isFull(widx, ridx uint32) bool {
	return widx-ridx >= r.cap-1
}

func (r *RingBufferWithoutAlign[T]) isEmpty(widx, ridx uint32) bool {
	return widx == ridx || widx < ridx
}
