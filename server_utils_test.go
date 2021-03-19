package gemax_test

import (
	"bytes"
	"context"
	"net/url"
	"testing"

	"github.com/ninedraft/gemax"
	"github.com/ninedraft/gemax/status"
)

func TestServeContent(test *testing.T) {
	var testdata = []byte("example text")
	var ctx = context.Background()
	var rw = &responseRecorder{}
	var req = &request{
		remoteAddr: test.Name(),
		url:        "gemini://localhost.example.net",
	}
	var serve = gemax.ServeContent("application/octet-stream", testdata)

	serve(ctx, rw, req)

	if rw.status != status.Success {
		test.Errorf("%s is expected, got %s", status.Success, rw.status)
	}
	if !bytes.Equal(rw.Bytes(), testdata) {
		test.Errorf("%q expected, got %q", testdata, rw)
	}
}

type request struct {
	remoteAddr string
	url        string
}

func (req *request) URL() *url.URL {
	var u, _ = url.Parse(req.url)
	return u
}

func (req *request) RemoteAddr() string {
	return req.remoteAddr
}

type responseRecorder struct {
	status status.Code
	meta   string
	bytes.Buffer
}

func (r *responseRecorder) Close() error {
	return nil
}

func (r *responseRecorder) WriteStatus(code status.Code, meta string) {
	if r.status != 0 {
		return
	}
	if code == status.Success && meta == "" {
		meta = gemax.MIMEGemtext
	}
	r.status = code
	r.meta = meta
}
