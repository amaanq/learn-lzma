package lzma

import "errors"

// minLC and maxLC define the range for LC values.
const (
	minLC = 0
	maxLC = 8
)

// minLP and maxLP define the range for LP values.
const (
	minLP = 0
	maxLP = 4
)

// maximum and minimum values for the LZMA properties.
const (
	minPB = 0
	maxPB = 4
)

type Properties struct {
	LC, LP, PB int
}

func (p *Properties) verify() error {
	if p == nil {
		return errors.New("lzma: properties are nil")
	}
	if p.LC < minLC || p.LC > maxLC {
		return errors.New("lzma: lc out of range")
	}
	if p.LP < minLP || p.LP > maxLP {
		return errors.New("lzma: lp out of range")
	}
	if p.PB < minPB || p.PB > maxPB {
		return errors.New("lzma: pb out of range")
	}
	return nil
}

func (p Properties) ToByte() byte {
	return byte((p.PB*5+p.LP)*9 + p.LC)
}
