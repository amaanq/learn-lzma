package lzma

type operation interface {
	Len() int
}

type match struct {
	distance int64
	n        int
}

func (m match) Len() int {
	return m.n
}

type lit struct {
	b byte
}

func (l lit) Len() int {
	return 1
}
