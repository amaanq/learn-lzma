package lzma

import (
	"encoding/binary"
	"fmt"
)

const noHeaderSize uint64 = 1<<64 - 1

type header struct {
	properties Properties
	dictCap    int
	size       int64
}

func (h *header) marshalBinary() ([]byte, error) {
	if err := h.properties.verify(); err != nil {
		return nil, err
	}
	if h.dictCap < 0 || int64(h.dictCap) > MaxDictCap {
		return nil, fmt.Errorf("lzma: DictCap %d out of range", h.dictCap)
	}

	data := make([]byte, 13)

	data[0] = h.properties.ToByte()

	binary.LittleEndian.PutUint32(data[1:5], uint32(h.dictCap))

	s := noHeaderSize
	if h.size > 0 {
		s = uint64(h.size)
	}
	binary.LittleEndian.PutUint64(data[5:], s)

	return data, nil
}
