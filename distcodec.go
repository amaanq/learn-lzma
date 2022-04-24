package lzma

const (
	// minimum supported distance
	minDistance = 1
	// maximum supported distance, value is used for the eos marker.
	maxDistance = 1 << 32
	// number of the supported len states
	lenStates = 4
	// start for the position models
	startPosModel = 4
	// first index with align bits support
	endPosModel = 14
	// bits for the position slots
	posSlotBits = 6
	// number of align bits
	alignBits = 4
)
