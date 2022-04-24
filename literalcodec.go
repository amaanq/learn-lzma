package lzma

type literalCodec struct {
	probs []prob
}

func (c *literalCodec) init(lc, lp int) {
	switch {
	case lc < minLC || lc > maxLC:
		panic("lc out of range")
	case lp < minLP || lp > maxLP:
		panic("lp out of range")
	}
	c.probs = make([]prob, 0x300<<uint(lc+lp))
	for i := range c.probs {
		c.probs[i] = probInit
	}
}

func (c *literalCodec) Encode(e *rangeEncoder, s byte, state uint32, match byte, litState uint32) error {
	k := litState * 0x300
	probs := c.probs[k : k+0x300]
	symbol := uint32(1)
	r := uint32(s)
	if state >= 7 {
		m := uint32(match)
		for {
			matchBit := (m >> 7) & 1
			m <<= 1
			bit := (r >> 7) & 1
			r <<= 1
			i := ((1 + matchBit) << 8) | symbol
			if err := probs[i].Encode(e, bit); err != nil {
				return err
			}
			symbol = (symbol << 1) | bit
			if matchBit != bit {
				break
			}
			if symbol >= 0x100 {
				break
			}
		}
	}
	for symbol < 0x100 {
		bit := (r >> 7) & 1
		r <<= 1
		if err := probs[symbol].Encode(e, bit); err != nil {
			return err
		}
		symbol = (symbol << 1) | bit
	}
	return nil
}
