package lzma

import (
	"errors"
	"io"
)

var ErrLimit = errors.New("limit reached")

type LimitedByteWriter struct {
	BW io.ByteWriter
	N  int64
}

func (l *LimitedByteWriter) WriteByte(c byte) error {
	if l.N <= 0 {
		return ErrLimit
	}
	if err := l.BW.WriteByte(c); err != nil {
		return err
	}
	l.N--
	return nil
}
