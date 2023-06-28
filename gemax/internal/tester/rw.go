package tester

import (
	"bytes"

	"github.com/ninedraft/gemax/gemax"
	"github.com/ninedraft/gemax/gemax/status"
)

var _ gemax.ResponseWriter = &ResponseWriter{}

type ResponseWriter struct {
	Status   status.Code
	Body     bytes.Buffer
	Meta     string
	IsClosed bool
}

func (rw *ResponseWriter) Write(b []byte) (n int, err error) {
	return rw.Body.Write(b)
}

func (rw *ResponseWriter) WriteStatus(code status.Code, meta string) {
	rw.Status = code
	rw.Meta = meta
}

func (rw *ResponseWriter) Close() error {
	rw.IsClosed = true
	return nil
}
