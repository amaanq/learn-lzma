package lzma

const movebits = 5

const probbits = 11

type prob uint16

func (p *prob) dec() {
	*p -= *p >> movebits
}

func (p *prob) inc() {
	*p += ((1 << probbits) - *p) >> movebits
}

func (p prob) bound(r uint32) uint32 {
	return (r >> probbits) * uint32(p)
}

func (p *prob) Encode(e *rangeEncoder, v uint32) error {
	return e.EncodeBit(v, p)
}
