package gemax_test

import (
	"bytes"
	"context"
	"crypto/x509"
	"io"
	"io/fs"
	"net/url"
	"testing"
	"testing/fstest"
	"time"

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

func TestFS_ReadDirError_NilLogger_NoPanic(test *testing.T) {
	test.Parallel()

	var fserve = gemax.FileSystem{
		FS: readDirErrorFS{},
	}
	var rw = &responseWriter{}
	var req = &incomingRequest{
		remoteAddr: test.Name(),
	}
	req.url, _ = url.Parse("gemini://example.com/blog")

	var panicked bool
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		fserve.Serve(context.Background(), rw, req)
	}()

	if panicked {
		test.Fatal("Serve panicked with nil logger")
	}
	if rw.status != status.ServerUnavailable {
		test.Fatalf("expected %q, got %q", status.ServerUnavailable, rw.status)
	}
}

type readDirErrorFS struct{}

func (fsys readDirErrorFS) Open(name string) (fs.File, error) {
	if name == "blog" {
		return &readDirErrorFile{}, nil
	}
	return nil, fs.ErrNotExist
}

type readDirErrorFile struct{}

func (file *readDirErrorFile) Stat() (fs.FileInfo, error) { return readDirErrorInfo{}, nil }
func (file *readDirErrorFile) Read([]byte) (int, error)   { return 0, io.EOF }
func (file *readDirErrorFile) Close() error               { return nil }
func (file *readDirErrorFile) ReadDir(int) ([]fs.DirEntry, error) {
	return nil, fs.ErrPermission
}

type readDirErrorInfo struct{}

func (info readDirErrorInfo) Name() string       { return "blog" }
func (info readDirErrorInfo) Size() int64        { return 0 }
func (info readDirErrorInfo) Mode() fs.FileMode  { return fs.ModeDir }
func (info readDirErrorInfo) ModTime() time.Time { return time.Time{} }
func (info readDirErrorInfo) IsDir() bool        { return true }
func (info readDirErrorInfo) Sys() any           { return nil }

type incomingRequest struct {
	url        *url.URL
	remoteAddr string
}

func (req *incomingRequest) RemoteAddr() string { return req.remoteAddr }

func (req *incomingRequest) URL() *url.URL { return req.url }

func (req *incomingRequest) Certificates() []*x509.Certificate {
	return nil
}

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
