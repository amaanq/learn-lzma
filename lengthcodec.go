package lzma

const maxPosBits = 4

const (
	minMatchLen = 2
	maxMatchLen = minMatchLen + 16 + 256 - 1
)
