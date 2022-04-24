package lzma

import (
	"bufio"
	"errors"
	"io"
)

// MinDictCap and MaxDictCap provide the range of supported dictionary
// capacities.
const (
	MinDictCap = 1 << 12
	MaxDictCap = 1<<32 - 1
)

type Writer struct {
	h   header
	bw  io.ByteWriter
	buf *bufio.Writer
	e   *encoder
}

type WriterConfig struct {
	Properties   *Properties
	DictCap      int
	BufSize      int
	Matcher      MatchAlgorithm
	SizeInHeader bool
	Size         int64
	EOSMarker    bool
}

func NewWriter(lzma io.Writer) (*Writer, error) {
	return WriterConfig{}.NewWriter(lzma)
}

func (c WriterConfig) NewWriter(lzma io.Writer) (*Writer, error) {
	if err := c.Verify(); err != nil {
		return nil, err
	}
	w := &Writer{h: c.header()}

	var ok bool
	w.bw, ok = lzma.(io.ByteWriter)
	if !ok {
		w.buf = bufio.NewWriter(lzma)
		w.bw = w.buf
	}
	state := newState(w.h.properties)
	m, err := c.Matcher.new(w.h.dictCap)
	if err != nil {
		return nil, err
	}
	dict, err := newEncoderDict(w.h.dictCap, c.BufSize, m)
	if err != nil {
		return nil, err
	}
	var flags encoderFlags
	if c.EOSMarker {
		flags = eosMarker
	}
	if w.e, err = newEncoder(w.bw, state, dict, flags); err != nil {
		return nil, err
	}

	if err = w.writeHeader(); err != nil {
		return nil, err
	}
	return w, nil
}

func (c *WriterConfig) fill() {
	if c.Properties == nil {
		c.Properties = &Properties{LC: 3, LP: 0, PB: 2}
	}
	if c.DictCap == 0 {
		c.DictCap = 8 * 1024 * 1024
	}
	if c.BufSize == 0 {
		c.BufSize = 4096
	}
	if c.Size > 0 {
		c.SizeInHeader = true
	}
	if !c.SizeInHeader {
		c.EOSMarker = true
	}
}

func (c *WriterConfig) Verify() error {
	c.fill() // init?
	if c == nil {
		return errors.New("lzma: WriterConfig is nil")
	}
	if c.Properties == nil {
		return errors.New("lzma: WriterConfig has no Properties set")
	}
	if err := c.Properties.verify(); err != nil {
		return err
	}
	if c.DictCap < MinDictCap || int64(c.DictCap) > MaxDictCap {
		return errors.New("lzma: dictionary capacity is out of range")
	}
	if c.BufSize < maxMatchLen {
		return errors.New("lzma: lookahead buffer size too small")
	}
	if c.SizeInHeader {
		if c.Size < 0 {
			return errors.New("lzma: negative size not supported")
		}
	} else if !c.EOSMarker {
		return errors.New("lzma: EOS marker is required")
	}
	if err := c.Matcher.verify(); err != nil {
		return err
	}

	return nil
}

func (c *WriterConfig) header() header {
	h := header{
		properties: *c.Properties,
		dictCap:    c.DictCap,
		size:       -1,
	}
	if c.SizeInHeader {
		h.size = c.Size
	}
	return h
}

func (w *Writer) writeHeader() error {
	data, err := w.h.marshalBinary()
	if err != nil {
		return err
	}
	_, err = w.bw.(io.Writer).Write(data)
	return err
}

func (w *Writer) Write(p []byte) (int, error) {
	var err error
	if w.h.size >= 0 {
		m := w.h.size
		m -= w.e.Compressed() + int64(w.e.dict.Buffered())
		if m < 0 {
			m = 0
		}
		if m < int64(len(p)) {
			p = p[:m]
			err = ErrNoSpace
		}
	}

	var n int
	var werr error
	if n, werr = w.e.Write(p); werr != nil {
		err = werr
	}
	return n, err
}

func (w *Writer) Close() error {
	if w.h.size >= 0 {
		n := w.e.Compressed() + int64(w.e.dict.Buffered())
		if n != w.h.size {
			return errSize 
		}
	}
	err := w.e.Close()
	if w.buf != nil {
		ferr := w.buf.Flush()
		if err == nil {
			err = ferr 
		}
	}
	return err 
}