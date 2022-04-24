package lzma

import (
	"errors"
	"fmt"
	"io"
)

type matcher interface {
	io.Writer
	SetDict(d *encoderDict)
	NextOp(rep [4]uint32) operation
}

type encoderDict struct {
	buf      buffer
	m        matcher
	head     int64
	capacity int
	data     [maxMatchLen]byte
}

func newEncoderDict(dictCap, bufSize int, m matcher) (*encoderDict, error) {
	if dictCap < 1 || int64(dictCap) > MaxDictCap {
		return nil, errors.New("lzma: dictionary capacity out of range")
	}
	if bufSize < 1 {
		return nil, errors.New("lzma: buffer sie must be larger than zero")
	}
	d := &encoderDict{
		buf:      *newBuffer(dictCap + bufSize),
		capacity: dictCap,
		m:        m,
	}
	m.SetDict(d)
	return d, nil
}

func (d *encoderDict) Discard(n int) {
	p := d.data[:n]
	k, _ := d.buf.Read(p)
	if k < n {
		panic(fmt.Errorf("lzma: can't discard %d bytes", n))
	}
	d.head += int64(n)
	d.m.Write(p)
}

func (d *encoderDict) Len() int {
	n := d.buf.Available()
	if int64(n) > d.head {
		return int(d.head)
	}
	return n
}

func (d *encoderDict) DictLen() int {
	if d.head < int64(d.capacity) {
		return int(d.head)
	}
	return d.capacity
}

func (d *encoderDict) Available() int {
	return d.buf.Available() - d.DictLen()
}

func (d *encoderDict) Write(p []byte) (int, error) {
	var err error
	m := d.Available()
	if len(p) > m {
		p = p[:m]
		err = ErrNoSpace
	}
	var n int
	var e error
	if n, e = d.buf.Write(p); e != nil {
		err = e
	}
	return n, err
}

func (d *encoderDict) Pos() int64 { return d.head }

func (d *encoderDict) ByteAt(distance int) byte {
	if distance <= 0 || distance > d.Len() {
		return 0
	}
	i := d.buf.rear - distance
	if i < 0 {
		i += len(d.buf.data)
	}
	return d.buf.data[i]
}

func (d *encoderDict) Buffered() int { return d.buf.Buffered() }
