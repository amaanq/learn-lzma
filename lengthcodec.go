package lzma

import "errors"

const maxPosBits = 4

const (
	minMatchLen = 2
	maxMatchLen = minMatchLen + 16 + 256 - 1
)

type lengthCodec struct {
	choice [2]prob
	low    [1 << maxPosBits]treeCodec
	mid    [1 << maxPosBits]treeCodec
	high   treeCodec
}

func (lc *lengthCodec) init() {
	for i := range lc.choice {
		lc.choice[i] = probInit
	}
	for i := range lc.low {
		lc.low[i] = makeTreeCodec(3)
	}
	for i := range lc.mid {
		lc.mid[i] = makeTreeCodec(3)
	}
	lc.high = makeTreeCodec(8)
}

func (lc *lengthCodec) Encode(e *rangeEncoder, l uint32, posState uint32) error {
	if l > maxMatchLen-minMatchLen {
		return errors.New("lengthCodec.Encode: l out of range")
	}
	if l < 8 {
		if err := lc.choice[0].Encode(e, 0); err != nil {
			return err
		}
		return lc.low[posState].Encode(e, l)
	}
	if err := lc.choice[0].Encode(e, 1); err != nil {
		return err
	}
	if l < 16 {
		if err := lc.choice[1].Encode(e, 0); err != nil {
			return err
		}
		return lc.mid[posState].Encode(e, l-8)
	}
	if err := lc.choice[1].Encode(e, 1); err != nil {
		return err
	}
	if err := lc.high.Encode(e, l-16); err != nil {
		return err
	}
	return nil
}
