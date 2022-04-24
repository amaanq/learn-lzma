package lzma

const states = 12

type state struct {
	rep         [4]uint32
	isMatch     [states << maxPosBits]prob
	isRepG0Long [states << maxPosBits]prob
	isRep       [states]prob
	isRepG0     [states]prob
	isRepG1     [states]prob
	isRepG2     [states]prob
	litCodec    literalCodec
	lenCodec    lengthCodec
	repLenCodec lengthCodec
	distCodec   distCodec
	state       uint32
	posBitMask  uint32
	Properties  Properties
}

func initProbSlice(p []prob) {
	for i := range p {
		p[i] = probInit
	}
}

func (s *state) Reset() {
	p := s.Properties
	*s = state{
		Properties: p,
		posBitMask: (uint32(1) << uint(p.PB)) - 1,
	}
	initProbSlice(s.isMatch[:])
	initProbSlice(s.isRep[:])
	initProbSlice(s.isRepG0[:])
	initProbSlice(s.isRepG1[:])
	initProbSlice(s.isRepG2[:])
	initProbSlice(s.isRepG0Long[:])
	s.litCodec.init(p.LC, p.LP)
	s.lenCodec.init()
	s.repLenCodec.init()
	s.distCodec.init()
}

func newState(p Properties) *state {
	s := &state{Properties: p}
	s.Reset()
	return s
}

func (s *state) updateStateLiteral() {
	switch {
	case s.state < 4:
		s.state = 0
		return
	case s.state < 10:
		s.state -= 3
		return
	}
	s.state -= 6
}

func (s *state) updateStateMatch() {
	if s.state < 7 {
		s.state = 7
	} else {
		s.state = 10
	}
}

func (s *state) updateStateRep() {
	if s.state < 7 {
		s.state = 8
	} else {
		s.state = 11
	}
}

func (s *state) updateStateShortRep() {
	if s.state < 7 {
		s.state = 9
	} else {
		s.state = 11
	}
}

func (s *state) states(dictHead int64) (uint32, uint32, uint32) {
	state1 := s.state
	posState := uint32(dictHead) & s.posBitMask
	state2 := (s.state << maxPosBits) | posState
	return state1, state2, posState
}

func (s *state) litState(prev byte, dictHead int64) uint32 {
	lp, lc := uint(s.Properties.LP), uint(s.Properties.LC)
	litState := ((uint32(dictHead) & ((1 << lp) - 1)) << lc) | (uint32(prev) >> (8 - lc))
	return litState
}
