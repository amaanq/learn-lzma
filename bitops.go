package lzma

const ntz32Const = 0x04d7651f

var ntz32Table = [32]int8{
	0, 1, 2, 24, 3, 19, 6, 25,
	22, 4, 20, 10, 16, 7, 12, 26,
	31, 23, 18, 5, 21, 9, 15, 11,
	30, 17, 8, 14, 29, 13, 28, 27,
}

func nlz32(x uint32) int {
	// Smear left most bit to the right
	x |= x >> 1
	x |= x >> 2
	x |= x >> 4
	x |= x >> 8
	x |= x >> 16
	// Use ntz mechanism to calculate nlz.
	x++
	if x == 0 {
		return 0
	}
	x *= ntz32Const
	return 32 - int(ntz32Table[x>>27])
}
