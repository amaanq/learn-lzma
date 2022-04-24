package lzma

import "errors"

var (
	errSize = errors.New("lzma: wrong uncompressed data size")
)
