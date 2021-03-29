// Package bufwriter provides a buffered writer gadget.
package bufwriter

import (
	"bufio"
	"errors"
	"io"

	"github.com/ninedraft/gemax/internal/multierr"
)

// Writer is a buffered io.Writer wrapper.
type Writer struct {
	isClosed bool
	closer   io.Closer
	*writer
}

// DefaultBufferSize is used if buffers size is <= 0.
const DefaultBufferSize = 16 << 10 // 16 Kb

// New creates a new buffered writer.
// If bufSize <= 0, then DefaultBufferSize is used as internal buffer size.
func New(w io.WriteCloser, bufSize int) *Writer {
	if bufSize <= 0 {
		bufSize = DefaultBufferSize
	}
	return &Writer{
		closer: w,
		writer: bufio.NewWriterSize(w, bufSize),
	}
}

type writer = bufio.Writer

// Reset buffer and sets new write target.
func (wr *Writer) Reset(w io.WriteCloser) {
	wr.isClosed = false
	wr.closer = w
	wr.writer.Reset(w)
}

var errAlreadyClosed = errors.New("already closed")

// Close flushes and closes underlying writer.
func (wr *Writer) Close() error {
	if wr.isClosed {
		return errAlreadyClosed
	}
	wr.isClosed = true
	return multierr.Combine(
		wr.Flush(),
		wr.closer.Close(),
	)
}
