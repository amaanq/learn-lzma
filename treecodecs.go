package lzma

type treeCodec struct {
	probTree
}

func makeTreeCodec(bits int) treeCodec {
	return treeCodec{makeProbTree(bits)}
}

func (tc *treeCodec) Encode(e *rangeEncoder, v uint32) error {
	m := uint32(1)
	for i := int(tc.bits) - 1; i >= 0; i-- {
		b := (v >> uint(i)) & 1
		if err := e.EncodeBit(b, &tc.probs[m]); err != nil {
			return err
		}
		m = (m << 1) | b
	}
	return nil
}

type treeReverseCodec struct {
	probTree
}

func makeTreeReverseCodec(bits int) treeReverseCodec {
	return treeReverseCodec{makeProbTree(bits)}
}

func (tc *treeReverseCodec) Encode(v uint32, e *rangeEncoder) error {
	m := uint32(1)
	for i := uint(0); i < uint(tc.bits); i++ {
		b := (v >> i) & 1
		if err := e.EncodeBit(b, &tc.probs[m]); err != nil {
			return err
		}
		m = (m << 1) | b
	}
	return nil
}

type probTree struct {
	probs []prob
	bits  byte
}

func makeProbTree(bits int) probTree {
	if bits < 1 || bits > 32 {
		panic("bits outside of range [1,32]")
	}
	t := probTree{
		bits:  byte(bits),
		probs: make([]prob, 1<<uint(bits)),
	}
	for i := range t.probs {
		t.probs[i] = probInit
	}
	return t
}

func (t *probTree) Bits() int {
	return int(t.bits)
}
