package bufreader

import (
	"bufio"
	"io"
)

type (
	Reader struct {
		closer
		*buf
	}

	buf    = bufio.Reader
	closer = io.Closer
)

const (
	DefaultMaxSize    = 1 << 20
	DefaultBufferSize = 16 << 10
)

func New(re io.ReadCloser, bufSize int, max int64) *Reader {
	if bufSize <= 0 {
		bufSize = DefaultBufferSize
	}
	if max <= 0 {
		max = DefaultMaxSize
	}
	var buf = bufio.NewReaderSize(io.LimitReader(re, max), bufSize)
	return &Reader{
		closer: re,
		buf:    buf,
	}
}

func (re *Reader) Reset(rc io.ReadCloser) {
	re.closer = rc
	re.buf.Reset(rc)
}
