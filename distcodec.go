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

type distCodec struct {
	posSlotCodecs [lenStates]treeCodec
	posModel      [endPosModel - startPosModel]treeReverseCodec
	alignCodec    treeReverseCodec
}

func (dc *distCodec) init() {
	for i := range dc.posSlotCodecs {
		dc.posSlotCodecs[i] = makeTreeCodec(posSlotBits)
	}
	for i := range dc.posModel {
		posSlot := startPosModel + i
		bits := (posSlot >> 1) - 1
		dc.posModel[i] = makeTreeReverseCodec(bits)
	}
	dc.alignCodec = makeTreeReverseCodec(alignBits)
}

func lenState(l uint32) uint32 {
	if l >= lenStates {
		l = lenStates - 1
	}
	return l
}

func (dc *distCodec) Encode(e *rangeEncoder, dist, l uint32) error {
	var posSlot uint32
	var bits uint32
	if dist < startPosModel {
		posSlot = dist
	} else {
		bits = uint32(30 - nlz32(dist))
		posSlot = startPosModel - 2 + (bits << 1)
		posSlot += (dist >> uint(bits)) & 1
	}

	if err := dc.posSlotCodecs[lenState(l)].Encode(e, posSlot); err != nil {
		return err
	}

	switch {
	case posSlot < startPosModel:
		return nil
	case posSlot < endPosModel:
		tc := &dc.posModel[posSlot-startPosModel]
		return tc.Encode(dist, e)
	}
	dic := directCodec(bits - alignBits)
	if err := dic.Encode(e, dist>>alignBits); err != nil {
		return err
	}
	return dc.alignCodec.Encode(dist, e)
}
