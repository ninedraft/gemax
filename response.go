package gemax

import (
	"fmt"
	"io"

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
	dst           io.WriteCloser
}

func (rw *responseWriter) WriteStatus(code status.Code, meta string) {
	if rw.statusWritten {
		return
	}
	if code.Class() == status.Success && meta == "" {
		meta = MIMEGemtext
	}
	_, _ = fmt.Fprintf(rw.dst, "%d %s\r\n", code, meta)
	rw.status = code
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.statusWritten {
		rw.WriteStatus(status.Success, MIMEGemtext)
	}
	return rw.dst.Write(data)
}

func (rw *responseWriter) Close() error {
	return rw.dst.Close()
}
