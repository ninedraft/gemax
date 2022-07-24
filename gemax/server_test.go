package gemax_test

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ninedraft/gemax/gemax"
	"github.com/ninedraft/gemax/gemax/internal/testaddr"
	"github.com/ninedraft/gemax/gemax/status"
)

func TestServerSuccess(test *testing.T) {
	var listener, server = setupEchoServer(test)
	server.Hosts = []string{"example.com"}
	defer func() { _ = listener.Close() }()
	var ctx, cancel = context.WithCancel(context.Background())
	test.Cleanup(cancel)
	runTask(test, func() {
		var err = server.Serve(ctx, listener)
		if err != nil {
			test.Logf("test server: Serve: %v", err)
		}
	})

	var resp = listener.next(test.Name(), strings.NewReader("gemini://example.com/path"))

	expectResponse(test, resp, "20 text/gemini\r\ngemini://example.com/path")
}

func TestServerBadRequest(test *testing.T) {
	var listener, server = setupEchoServer(test)
	defer func() { _ = listener.Close() }()
	var ctx, cancel = context.WithCancel(context.Background())
	test.Cleanup(cancel)
	runTask(test, func() {
		var err = server.Serve(ctx, listener)
		if err != nil {
			test.Logf("test server: Serve: %v", err)
		}
	})

	var resp = listener.next(test.Name(), strings.NewReader("invalid URL"))

	expectResponse(test, resp, "59 "+status.Text(status.BadRequest)+"\r\n")
}

func TestServerInvalidHost(test *testing.T) {
	var listener, server = setupEchoServer(test)
	server.Hosts = []string{"example.com"}
	defer func() { _ = listener.Close() }()
	var ctx, cancel = context.WithCancel(context.Background())
	test.Cleanup(cancel)
	runTask(test, func() {
		var err = server.Serve(ctx, listener)
		if err != nil {
			test.Logf("test server: Serve: %v", err)
		}
	})

	var resp = listener.next(test.Name(), strings.NewReader("gemini://another.com/path"))

	expectResponse(test, resp, "50 host not found\r\n")
}

func TestServerCancelListen(test *testing.T) {
	var server = &gemax.Server{
		Addr: testaddr.Addr(),
		Logf: test.Logf,
		Handler: func(ctx context.Context, rw gemax.ResponseWriter, req gemax.IncomingRequest) {
			_, _ = io.WriteString(rw, "example text")
		},
	}
	test.Logf("loading test certs")
	var cert, errCert = tls.LoadX509KeyPair("testdata/cert.pem", "testdata/key.pem")
	if errCert != nil {
		test.Fatal(errCert)
	}
	var cfg = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}
	var ctx, cancel = context.WithCancel(context.Background())
	test.Cleanup(cancel)
	test.Logf("starting test server")
	test.Logf("test server: listening on %q", server.Addr)

	time.AfterFunc(100*time.Millisecond, cancel)
	var err = server.ListenAndServe(ctx, cfg)
	if err != nil {
		test.Errorf("unexpected error %v, while nil is expected", err)
	}
}

func TestListenAndServe(test *testing.T) {
	var server = &gemax.Server{
		Addr: "localhost:40423",
		Logf: test.Logf,
		Handler: func(ctx context.Context, rw gemax.ResponseWriter, req gemax.IncomingRequest) {
			_, _ = io.WriteString(rw, "example text")
		},
	}
	test.Logf("loading test certs")
	var cert, errCert = tls.LoadX509KeyPair("testdata/cert.pem", "testdata/key.pem")
	if errCert != nil {
		test.Fatal(errCert)
	}
	var cfg = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}
	var ctx, cancel = context.WithCancel(context.Background())
	test.Cleanup(cancel)
	test.Logf("starting test server")
	go func() {
		test.Logf("test server: listening on %q", server.Addr)
		var err = server.ListenAndServe(ctx, cfg)
		if err != nil {
			test.Logf("test server: Serve: %v", err)
		}
	}()
	time.Sleep(time.Second)
	var client = &gemax.Client{}
	var resp, errFetch = client.Fetch(ctx, "gemini://"+server.Addr)
	if errFetch != nil {
		test.Error("fetching: ", errFetch)
		return
	}
	defer func() { _ = resp.Close() }()

	expectResponse(test, resp, "example text")
	var data, errRead = io.ReadAll(resp)
	test.Logf("%s / %v", data, errRead)
}

// emulates michael-lazar/gemini-diagnostics localhost $PORT --checks='URLDotEscape'
func TestURLDotEscape(test *testing.T) {
	var listener, server = setupEchoServer(test)
	server.Hosts = []string{"example.com"}
	defer func() { _ = listener.Close() }()
	var ctx, cancel = context.WithCancel(context.Background())
	test.Cleanup(cancel)
	runTask(test, func() {
		var err = server.Serve(ctx, listener)
		if err != nil {
			test.Logf("test server: Serve: %v", err)
		}
	})

	var resp = listener.next(test.Name(), strings.NewReader("gemini://example.com/../../\r\n"))

	expectResponse(test, resp, "50 50 PERMANENT FAILURE\r\n")
}

// emulates michael-lazar/gemini-diagnostics localhost 9999 --checks='PageNotFound'
func TestPageNotFound(test *testing.T) {
	test.Run("helper", func(test *testing.T) {
		var listener, server = setupServer(test,
			func(_ context.Context, rw gemax.ResponseWriter, req gemax.IncomingRequest) {
				gemax.NotFound(rw, req)
			})
		server.Hosts = []string{"example.com"}
		defer func() { _ = listener.Close() }()
		var ctx, cancel = context.WithCancel(context.Background())
		test.Cleanup(cancel)
		runTask(test, func() {
			var err = server.Serve(ctx, listener)
			if err != nil {
				test.Logf("test server: Serve: %v", err)
			}
		})

		var resp = listener.next(test.Name(), strings.NewReader("gemini://example.com/notexist\r\n"))

		expectResponse(test, resp, "51 gemini://example.com/notexist is not found\r\n")
	})

	test.Run("custom", func(test *testing.T) {
		test.Log("meta must not interfere with response body")
		var listener, server = setupServer(test,
			func(_ context.Context, rw gemax.ResponseWriter, req gemax.IncomingRequest) {
				rw.WriteStatus(status.NotFound, "page is not found\r\ndotdot")
			})
		server.Hosts = []string{"example.com"}
		defer func() { _ = listener.Close() }()
		var ctx, cancel = context.WithCancel(context.Background())
		test.Cleanup(cancel)
		runTask(test, func() {
			var err = server.Serve(ctx, listener)
			if err != nil {
				test.Logf("test server: Serve: %v", err)
			}
		})

		var resp = listener.next(test.Name(), strings.NewReader("gemini://example.com/notexist\r\n"))

		expectResponse(test, resp, "51 page is not found\tdotdot\r\n")
	})
}

func setupServer(t *testing.T, handler gemax.Handler) (*fakeListener, *gemax.Server) {
	t.Helper()
	var server = &gemax.Server{
		Logf:    t.Logf,
		Handler: handler,
	}
	var listener = newListener(t.Name())
	return listener, server
}

func setupEchoServer(t *testing.T) (*fakeListener, *gemax.Server) {
	t.Helper()
	return setupServer(t, func(ctx context.Context, rw gemax.ResponseWriter, req gemax.IncomingRequest) {
		_, _ = rw.Write([]byte(req.URL().String()))
	})
}

func expectResponse(t *testing.T, got io.Reader, want string) {
	t.Helper()
	var data, err = io.ReadAll(got)
	if err != nil {
		t.Fatal("unexpected error while reading response: ", err)
	}
	if string(data) != want {
		t.Fatalf("expected %q, got %q", want, data)
	}
}

type fakeListener struct {
	conns chan *fakeConn
	addr  string
}

func newListener(addr string) *fakeListener {
	return &fakeListener{
		addr:  addr,
		conns: make(chan *fakeConn),
	}
}

func (listener *fakeListener) next(addr string, data io.Reader) io.Reader {
	var pipe = newPipe()
	listener.conns <- &fakeConn{
		addr:        addr,
		localAddr:   addr,
		Reader:      data,
		WriteCloser: pipe,
	}
	return pipe
}

func (listener *fakeListener) Close() error {
	close(listener.conns)
	return nil
}

func (listener *fakeListener) Accept() (net.Conn, error) {
	var conn, ok = <-listener.conns
	if !ok {
		return nil, fmt.Errorf("listener closed: %w", io.EOF)
	}
	return conn, nil
}

func (listener *fakeListener) Addr() net.Addr {
	return fakeAddr(listener.addr)
}

type fakeConn struct {
	addr      string
	localAddr string
	io.Reader
	io.WriteCloser
}

func (conn *fakeConn) RemoteAddr() net.Addr {
	return fakeAddr(conn.addr)
}

func (conn *fakeConn) LocalAddr() net.Addr {
	return fakeAddr(conn.localAddr)
}

func (conn *fakeConn) SetDeadline(t time.Time) error {
	return nil
}

func (conn *fakeConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (conn *fakeConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type fakeAddr string

func (fakeAddr) Network() string { return "fake network" }

func (addr fakeAddr) String() string { return string(addr) }

func runTask(t *testing.T, task func()) {
	var done = make(chan struct{})
	go func() {
		defer close(done)
		task()
	}()
	t.Cleanup(func() {
		<-done
	})
}

type chPipe struct {
	closed bool
	ch     chan byte
}

func newPipe() *chPipe {
	return &chPipe{
		ch: make(chan byte),
	}
}

func (p *chPipe) Read(dst []byte) (int, error) {
	for i := range dst {
		var b, ok = <-p.ch
		if !ok {
			return i, io.EOF
		}
		dst[i] = b
	}
	return len(dst), nil
}

func (p *chPipe) Write(data []byte) (int, error) {
	for _, b := range data {
		p.ch <- b
	}
	return len(data), nil
}

var errAlreadyClosed = errors.New("already closed")

func (p *chPipe) Close() error {
	if p.closed {
		return errAlreadyClosed
	}
	close(p.ch)
	p.closed = true
	return nil
}
