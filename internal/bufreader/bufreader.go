// Package bufreader provides buffered reader-closer gadget.
package bufreader

import (
	"bufio"
	"io"
)

type (
	// Reader is a buffered reader-closer.
	// It can be reused.
	Reader struct {
		closer
		*buf
	}

	buf    = bufio.Reader
	closer = io.Closer
)

const (
	// DefaultBufferSize is used if bufSize is <= 0.
	DefaultBufferSize = 16 << 10
)

// New creates a new buffered reader.
func New(re io.ReadCloser, bufSize int) *Reader {
	if bufSize <= 0 {
		bufSize = DefaultBufferSize
	}
	var buf = bufio.NewReaderSize(re, bufSize)
	return &Reader{
		closer: re,
		buf:    buf,
	}
}

// Reset sets internal state and forces Reader to use provided reader.
// Provided reader can be nil.
func (re *Reader) Reset(rc io.ReadCloser) {
	re.closer = rc
	re.buf.Reset(rc)
}
