package lzma

import (
	"errors"
	"fmt"
	"mylzma/internal/hash"
)

const maxMatches = 16

const shortDists = 8

const (
	minTableExponent = 9
	maxTableExponent = 20
)

var newRoller = func(n int) hash.Roller { return hash.NewCyclicPoly(n) }

type hashTable struct {
	dict *encoderDict

	t         []int64
	data      []uint32
	front     int
	mask      uint64
	hoff      int64
	wordLen   int
	wr        hash.Roller
	hr        hash.Roller
	p         [maxMatches]int64
	distances [maxMatches + shortDists]int
}

func hashTableExponent(n uint32) int {
	e := 30 - nlz32(n)
	switch {
	case e < minTableExponent:
		e = minTableExponent
	case e > maxTableExponent:
		e = maxTableExponent
	}
	return e
}

func newHashTable(capacity int, wordLen int) (*hashTable, error) {
	if capacity <= 0 {
		return nil, errors.New("newHashTable: capacity must not be negative")
	}
	if wordLen < 1 || wordLen > 4 {
		return nil, errors.New("newHashTable: " + "argument wordLen out of range")
	}
	exp := hashTableExponent(uint32(capacity))
	n := 1 << uint(exp)
	if n <= 0 {
		panic("newHashTable: exponent is too large")
	}
	t := &hashTable{
		t:       make([]int64, n),
		data:    make([]uint32, capacity),
		mask:    (uint64(1) << uint(exp)) - 1,
		hoff:    -int64(wordLen),
		wordLen: wordLen,
		wr:      newRoller(wordLen),
		hr:      newRoller(wordLen),
	}
	return t, nil
}

func (t *hashTable) SetDict(d *encoderDict) { t.dict = d }

func (t *hashTable) buffered() int {
	n := t.hoff + 1
	switch {
	case n <= 0:
		return 0
	case n > int64(len(t.data)):
		return len(t.data)
	}
	return int(n)
}

func (t *hashTable) addIndex(i, n int) int {
	i += n - len(t.data)
	if i < 0 {
		i += len(t.data)
	}
	return i
}

func (t *hashTable) putDelta(delta uint32) {
	t.data[t.front] = delta
	t.front = t.addIndex(t.front, 1)
}

func (t *hashTable) putEntry(h uint64, pos int64) {
	if pos < 0 {
		return
	}
	i := h & t.mask
	old := t.t[i] - 1
	t.t[i] = pos + 1
	var delta int64
	if old >= 0 {
		delta = pos - old
		if delta > 1<<32-1 || delta > int64(t.buffered()) {
			delta = 0
		}
	}
	t.putDelta(uint32(delta))
}

func (t *hashTable) WriteByte(b byte) error {
	h := t.wr.RollByte(b)
	t.hoff++
	t.putEntry(h, t.hoff)
	return nil
}

func (t *hashTable) Write(p []byte) (n int, err error) {
	for _, b := range p {
		// WriteByte doesn't generate an error.
		t.WriteByte(b)
	}
	return len(p), nil
}

func (t *hashTable) getMatches(h uint64, positions []int64) int {
	if t.hoff < 0 || len(positions) == 0 {
		return 0
	}
	var n int
	buffered := t.buffered()
	tailPos := t.hoff + 1 - int64(buffered)
	rear := t.front - buffered
	if rear >= 0 {
		rear -= len(t.data)
	}

	pos := t.t[h&t.mask] - 1
	delta := pos - tailPos
	for {
		if delta < 0 {
			return n
		}
		positions[n] = tailPos + delta
		n++
		if n >= len(positions) {
			return n
		}
		i := rear + int(delta)
		if i < 0 {
			i += len(t.data)
		}
		u := t.data[i]
		if u == 0 {
			return n
		}
		delta -= int64(u)
	}
}

func (t *hashTable) hash(p []byte) uint64 {
	var h uint64
	for _, b := range p {
		h = t.hr.RollByte(b)
	}
	return h
}

func (t *hashTable) Matches(p []byte, positions []int64) int {
	if len(p) != t.wordLen {
		panic(fmt.Errorf("byte slice must have length %d", t.wordLen))
	}
	h := t.hash(p)
	return t.getMatches(h, positions)
}

func (t *hashTable) NextOp(rep [4]uint32) operation {
	data := t.dict.data[:maxMatchLen]
	n, _ := t.dict.buf.Peek(data)
	data = data[:n]
	var p []int64
	if n < t.wordLen {
		p = t.p[:0]
	} else {
		p = t.p[:maxMatches]
		n = t.Matches(data[:t.wordLen], p)
		p = p[:n]
	}

	head := t.dict.head
	dists := append(t.distances[:0], 1, 2, 3, 4, 5, 6, 7, 8)
	for _, pos := range p {
		dis := int(head - pos)
		if dis > shortDists {
			dists = append(dists, dis)
		}
	}

	var m match
	dictLen := t.dict.DictLen()
	for _, dist := range dists {
		if dist > dictLen {
			continue
		}

		i := t.dict.buf.rear - dist + m.n
		if i < 0 {
			i += len(t.dict.buf.data)
		}
		if t.dict.buf.data[i] != data[m.n] {
			continue
		}

		n := t.dict.buf.matchLen(dist, data)
		switch n {
		case 0:
			continue
		case 1:
			if uint32(dist-minDistance) != rep[0] {
				continue
			}
		}
		if n > m.n {
			m = match{int64(dist), n}
			if n == len(data) {
				break
			}
		}
	}

	if m.n == 0 {
		return lit{data[0]}
	}
	return m
}
