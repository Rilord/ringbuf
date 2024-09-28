package ringbuffer

import (
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/cpu"
)

const CacheLinePadSize = unsafe.Sizeof(cpu.CacheLinePad{})

type RingBuffer[T any] struct {
	cap   uint32
	mask  uint32
	_     [CacheLinePadSize - 8]byte
	write uint32
	read  uint32
	_     [CacheLinePadSize - 8]byte
	data  []*T
}

func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{
		cap:   uint32(capacity),
		mask:  uint32(capacity - 1),
		write: 0,
		read:  0,
		data:  make([]*T, capacity),
	}
}

func (r *RingBuffer[T]) Write(value T) (ok bool) {
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

func (r *RingBuffer[T]) Read() (val T, ok bool) {
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

func (r *RingBuffer[T]) ReadVec(buf []T) (read uint32) {
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

func (r *RingBuffer[T]) isFull(widx, ridx uint32) bool {
	return widx-ridx >= r.cap-1
}

func (r *RingBuffer[T]) isEmpty(widx, ridx uint32) bool {
	return widx == ridx || widx < ridx
}
