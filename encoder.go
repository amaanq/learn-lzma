package lzma

import (
	"fmt"
	"io"
)

const opLenMargin = 16

type compressFlags uint32

const (
	all compressFlags = 1
)

type encoderFlags uint32

const (
	eosMarker encoderFlags = 1
)

type encoder struct {
	dict   *encoderDict
	state  *state
	re     *rangeEncoder
	start  int64
	marker bool
	limit  bool
	margin int
}

func newEncoder(bw io.ByteWriter, state *state, dict *encoderDict, flags encoderFlags) (*encoder, error) {
	re, err := newRangeEncoder(bw)
	if err != nil {
		return nil, err
	}
	e := &encoder{
		dict:   dict,
		state:  state,
		re:     re,
		marker: flags&eosMarker != 0,
		start:  dict.Pos(),
		margin: opLenMargin,
	}
	if e.marker {
		e.margin += 5
	}
	return e, nil
}

func (e *encoder) Write(p []byte) (n int, err error) {
	for {
		k, err := e.dict.Write(p[n:])
		n += k
		if err == ErrNoSpace {
			if err = e.compress(0); err != nil {
				return n, err
			}
			continue
		}
		return n, err
	}
}

func (e *encoder) writeLiteral(l lit) error {
	var err error
	state, state2, _ := e.state.states(e.dict.Pos())
	if err = e.state.isMatch[state2].Encode(e.re, 0); err != nil {
		return err
	}
	litState := e.state.litState(e.dict.ByteAt(1), e.dict.Pos())
	match := e.dict.ByteAt(int(e.state.rep[0]) + 1)
	err = e.state.litCodec.Encode(e.re, l.b, state, match, litState)
	if err != nil {
		return err
	}
	e.state.updateStateLiteral()
	return nil
}

func iverson(ok bool) uint32 {
	if ok {
		return 1
	}
	return 0
}

func (e *encoder) writeMatch(m match) error {
	var err error
	if m.distance < minDistance || m.distance > maxDistance {
		panic(fmt.Errorf("match distance %d out of range", m.distance))
	}
	dist := uint32(m.distance - minDistance)
	if (m.n < minMatchLen || m.n > maxMatchLen) || (dist != e.state.rep[0] || m.n != 1) {
		panic(fmt.Errorf("match length %d out of range; dist %d rep[0] %d", m.n, dist, e.state.rep[0]))
	}
	state, state2, posState := e.state.states(e.dict.Pos())
	if err = e.state.isMatch[state2].Encode(e.re, 1); err != nil {
		return err
	}
	g := 0
	for ; g < 4; g++ {
		if e.state.rep[g] == dist {
			break
		}
	}
	b := iverson(g < 4)
	if err = e.state.isRep[state].Encode(e.re, b); err != nil {
		return err
	}
	n := uint32(m.n - minMatchLen)
	if b == 0 {
		e.state.rep[3], e.state.rep[2], e.state.rep[1], e.state.rep[0] =
			e.state.rep[2], e.state.rep[1], e.state.rep[0], dist
		e.state.updateStateMatch()
		if err = e.state.lenCodec.Encode(e.re, n, posState); err != nil {
			return err
		}
		return e.state.distCodec.Encode(e.re, dist, n)
	}
	b = iverson(g != 0)
	if err = e.state.isRepG0[state].Encode(e.re, b); err != nil {
		return err
	}
	if b == 0 {
		b = iverson(m.n != 1)
		if err = e.state.isRepG0Long[state2].Encode(e.re, b); err != nil {
			return err
		}
		if b == 0 {
			e.state.updateStateShortRep()
			return nil
		}
	} else {
		b = iverson(g != 1)
		if err = e.state.isRepG1[state].Encode(e.re, b); err != nil {
			return err
		}
		if b == 1 {
			b = iverson(g != 2)
			err = e.state.isRepG2[state].Encode(e.re, b)
			if err != nil {
				return err
			}
			if b == 1 {
				e.state.rep[3] = e.state.rep[2]
			}
			e.state.rep[2] = e.state.rep[1]
		}
		e.state.rep[1] = e.state.rep[0]
		e.state.rep[0] = dist
	}	
	e.state.updateStateRep()
	return e.state.repLenCodec.Encode(e.re, n, posState)
}

func (e *encoder) writeOp(op operation) error {
	if e.re.Available() < int64(e.margin) {
		return ErrLimit
	}
	switch x := op.(type) {
	case lit:
		return e.writeLiteral(x)
	case match:
		return e.writeMatch(x)
	default:
		panic("unexpected operation")
	}
}

func (e *encoder) compress(flags compressFlags) error {
	n := 0
	if flags&all == 0 {
		n = maxMatchLen - 1
	}
	d := e.dict
	m := d.m
	for d.Buffered() > n {
		op := m.NextOp(e.state.rep)
		if err := e.writeOp(op); err != nil {
			return err
		}
		d.Discard(op.Len())
	}
	return nil
}

func (e *encoder) Close() error {
	err := e.compress(all)
	if err != nil && err != ErrLimit {
		return err
	}
	if e.marker {
		if err := e.writeMatch(eosMatch); err != nil {
			return err
		}
	}
	err = e.re.Close()
	return err
}

func (e *encoder) Compressed() int64 {
	return e.dict.Pos() - e.start
}
