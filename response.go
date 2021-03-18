package gemax

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ninedraft/gemax/internal/multierr"

	"github.com/ninedraft/gemax/status"
)

// ResponseWriter describes a server side response writer.
type ResponseWriter interface {
	WriteStatus(code status.Code, meta string)
	io.WriteCloser
}

type responseWriter struct {
	status        status.Code
	statusWritten bool
	isClosed      bool
	dst           *bufferedWriter
}

func newResponseWriter(wr io.WriteCloser) *responseWriter {
	return &responseWriter{
		dst:      newBufferedWriter(wr),
		isClosed: false,
	}
}

func (rw *responseWriter) WriteStatus(code status.Code, meta string) {
	if rw.statusWritten || rw.isClosed {
		return
	}
	if code == status.Success && meta == "" {
		meta = MIMEGemtext
	}
	_, _ = fmt.Fprintf(rw.dst, "%d %s\r\n", code, meta)
	rw.status = code
	rw.statusWritten = true
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if rw.isClosed {
		return 0, io.ErrNoProgress
	}
	rw.WriteStatus(status.Success, MIMEGemtext)
	return rw.dst.Write(data)
}

func (rw *responseWriter) WriteString(s string) (int, error) {
	if rw.isClosed {
		return 0, io.ErrNoProgress
	}
	rw.WriteStatus(status.Success, MIMEGemtext)
	return rw.dst.WriteString(s)
}

var errAlreadyClosed = errors.New("already closed")

func (rw *responseWriter) Close() error {
	if rw.isClosed {
		return errAlreadyClosed
	}
	rw.WriteStatus(status.Success, MIMEGemtext)
	rw.isClosed = true
	var errClose = rw.dst.Close()
	putBufferedWriter(rw.dst)
	rw.dst = nil
	return errClose
}

type bufferedWriter struct {
	closer io.Closer
	*bufio.Writer
}

func (wr *bufferedWriter) Close() error {
	return multierr.Combine(
		wr.Writer.Flush(),
		wr.closer.Close(),
	)
}

const writeBufferSize = 4 * 1024

var bufioWriterPool = &sync.Pool{
	New: func() interface{} {
		return bufio.NewWriterSize(nil, writeBufferSize)
	},
}

func newBufferedWriter(wr io.WriteCloser) *bufferedWriter {
	var bwr = bufioWriterPool.Get().(*bufio.Writer)
	bwr.Reset(wr)
	return &bufferedWriter{
		closer: wr,
		Writer: bwr,
	}
}

func putBufferedWriter(bwr *bufferedWriter) {
	var wr = bwr.Writer
	bwr.Writer = nil
	bwr.closer = nil
	wr.Reset(nil)
	bufioWriterPool.Put(wr)
}
