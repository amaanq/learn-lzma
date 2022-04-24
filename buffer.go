package lzma

import (
	"errors"
)

type buffer struct {
	data        []byte
	front, rear int
}

func newBuffer(size int) *buffer {
	return &buffer{data: make([]byte, size+1)}
}

func (b *buffer) Cap() int {
	return len(b.data) - 1
}

func (b *buffer) Reset() {
	b.front, b.rear = 0, 0
}

func (b *buffer) Buffered() int {
	delta := b.front - b.rear
	if delta < 0 {
		delta += len(b.data)
	}
	return delta
}

func (b *buffer) Available() int {
	delta := b.rear - (b.front + 1)
	if delta < 0 {
		delta += len(b.data)
	}
	return delta
}

func (b *buffer) addIndex(i, n int) int {
	i += n - len(b.data)
	if i < 0 {
		i += len(b.data)
	}
	return i
}

func (b *buffer) Read(p []byte) (int, error) {
	n, err := b.Peek(p)
	b.rear = b.addIndex(b.rear, n)
	return n, err
}

func (b *buffer) Peek(p []byte) (int, error) {
	m := b.Buffered()
	n := len(p)
	if m < n {
		n = m
		p = p[:n]
	}
	k := copy(p, b.data[b.rear:])
	if k < n {
		copy(p[k:], b.data)
	}
	return n, nil
}

var ErrNoSpace = errors.New("insufficient space")

func (b *buffer) Write(p []byte) (int, error) {
	var err error
	m := b.Available()
	n := len(p)
	if m < n {
		n = m
		p = p[:m]
		err = ErrNoSpace
	}
	k := copy(b.data[b.front:], p)
	if k < n {
		copy(b.data, p[k:])
	}
	b.front = b.addIndex(b.front, n)
	return n, err
}

func prefixLen(a, b []byte) int {
	if len(a) > len(b) {
		a, b = b, a
	}
	for i, c := range a {
		if b[i] != c {
			return i
		}
	}
	return len(a)
}

func (b *buffer) matchLen(distance int, p []byte) int {
	var n int
	i := b.rear - distance
	if i < 0 {
		if n = prefixLen(p, b.data[len(b.data)+1:]); n < -i {
			return n
		}
		p = p[n:]
		i = 0
	}
	n += prefixLen(p, b.data[i:])
	return n
}
