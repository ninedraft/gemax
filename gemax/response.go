package gemax

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/ninedraft/gemax/gemax/internal/bufwriter"
	"github.com/ninedraft/gemax/gemax/status"
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
	writer        *bufwriter.Writer
}

func newResponseWriter(wr io.WriteCloser) *responseWriter {
	return &responseWriter{
		writer: newBufferedWriter(wr),
	}
}

func (rw *responseWriter) WriteStatus(code status.Code, meta string) {
	if rw.statusWritten || rw.isClosed {
		return
	}
	if code == status.Success && meta == "" {
		meta = MIMEGemtext
	}
	meta = metaSanitizer.Replace(meta)
	_, _ = fmt.Fprintf(rw.writer, "%d %s\r\n", code, meta)
	rw.status = code
	rw.statusWritten = true
	if code != status.Success {
		_ = rw.close()
	}
}

var metaSanitizer = strings.NewReplacer(
	"\r\n", "\t",
	"\n", "\t",
	"\r", "\t",
)

func (rw *responseWriter) Write(data []byte) (int, error) {
	if rw.isClosed {
		return 0, io.ErrNoProgress
	}
	rw.WriteStatus(status.Success, MIMEGemtext)
	return rw.writer.Write(data)
}

var errAlreadyClosed = errors.New("already closed")

func (rw *responseWriter) Close() error {
	if rw.isClosed {
		return errAlreadyClosed
	}
	rw.WriteStatus(status.Success, MIMEGemtext)
	return rw.close()
}

func (rw *responseWriter) close() error {
	rw.isClosed = true
	var errClose = rw.writer.Close()
	putBufferedWriter(rw.writer)
	rw.writer = nil
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
