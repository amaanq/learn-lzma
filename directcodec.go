package lzma

type directCodec byte

func (dc directCodec) Encode(e *rangeEncoder, v uint32) error {
	for i := int(dc - 1); i >= 0; i-- {
		if err := e.DirectEncodeBit(v >> uint(i)); err != nil {
			return err
		}
	}
	return nil
}
