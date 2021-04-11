package gemax_test

import (
	"bytes"
	"context"
	"net/url"
	"testing"
	"testing/fstest"

	"github.com/ninedraft/gemax/gemax"
	"github.com/ninedraft/gemax/gemax/status"
)

func TestFS(test *testing.T) {
	var fsys = fstest.MapFS{
		"gemax/index.gmi": {Data: []byte("# hello\n")},
		"map.gmi":         {Data: []byte("# hello\n")},
	}
	var fserve = gemax.FileSystem{
		FS:   fsys,
		Logf: test.Logf,
	}
	var ctx = context.Background()

	test.Run("", func(test *testing.T) {
		var rw = &responseWriter{}
		var req = &incomingRequest{
			remoteAddr: test.Name(),
		}
		req.url, _ = url.Parse("gemini://example.com")
		fserve.Serve(ctx, rw, req)
		if rw.status != status.Success {
			test.Errorf("expected %q, got %q", status.Success, rw.status)
		}
	})
}

type incomingRequest struct {
	url        *url.URL
	remoteAddr string
}

func (req *incomingRequest) RemoteAddr() string { return req.remoteAddr }

func (req *incomingRequest) URL() *url.URL { return req.url }

type responseWriter struct {
	status status.Code
	meta   string
	b      bytes.Buffer
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if rw.status == 0 {
		rw.status = status.Success
		rw.meta = gemax.MIMEGemtext
	}
	return rw.b.Write(data)
}

func (rw *responseWriter) WriteStatus(code status.Code, meta string) {
	if rw.status != 0 {
		return
	}
	rw.status = code
	rw.meta = meta
}

func (rw *responseWriter) Close() error {
	if rw.status == 0 {
		rw.status = status.Success
		rw.meta = gemax.MIMEGemtext
	}
	return nil
}
