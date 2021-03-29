package gemax

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ninedraft/gemax/internal/bufwriter"
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
	*bufwriterAlias
}

type bufwriterAlias = bufwriter.Writer

func newResponseWriter(wr io.WriteCloser) *responseWriter {
	return &responseWriter{
		bufwriterAlias: newBufferedWriter(wr),
	}
}

func (rw *responseWriter) WriteStatus(code status.Code, meta string) {
	if rw.statusWritten || rw.isClosed {
		return
	}
	if code == status.Success && meta == "" {
		meta = MIMEGemtext
	}
	_, _ = fmt.Fprintf(rw.bufwriterAlias, "%d %s\r\n", code, meta)
	rw.status = code
	rw.statusWritten = true
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if rw.isClosed {
		return 0, io.ErrNoProgress
	}
	rw.WriteStatus(status.Success, MIMEGemtext)
	return rw.bufwriterAlias.Write(data)
}

var errAlreadyClosed = errors.New("already closed")

func (rw *responseWriter) Close() error {
	if rw.isClosed {
		return errAlreadyClosed
	}
	rw.WriteStatus(status.Success, MIMEGemtext)
	rw.isClosed = true
	var errClose = rw.bufwriterAlias.Close()
	putBufferedWriter(rw.bufwriterAlias)
	rw.bufwriterAlias = nil
	return errClose
}

const writeBufferSize = 4 * 1024

var bufioWriterPool = &sync.Pool{
	New: func() interface{} {
		return bufwriter.New(nil, writeBufferSize)
	},
}

func newBufferedWriter(wr io.WriteCloser) *bufwriter.Writer {
	var bwr = bufioWriterPool.Get().(*bufwriter.Writer)
	bwr.Reset(wr)
	return bwr
}

func putBufferedWriter(bwr *bufwriter.Writer) {
	bwr.Reset(nil)
	bufioWriterPool.Put(bwr)
}
