package gemax_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ninedraft/gemax"
	"github.com/ninedraft/gemax/status"
)

func TestServerSuccess(test *testing.T) {
	var listener = setupEchoServer(test)
	defer func() { _ = listener.Close() }()

	var resp = listener.next(test.Name(), strings.NewReader("gemini://example.com/path"))

	expectResponse(test, resp, "20 text/gemini\r\ngemini://example.com/path")
}

func TestServerBadRequest(test *testing.T) {
	var listener = setupEchoServer(test)
	defer func() { _ = listener.Close() }()

	var resp = listener.next(test.Name(), strings.NewReader("invalid URL"))
	expectResponse(test, resp, "59 "+status.Text(status.BadRequest)+"\r\n")
}

func setupEchoServer(t *testing.T) *fakeListener {
	t.Helper()
	var server = &gemax.Server{
		Logf: t.Logf,
		Handler: func(ctx context.Context, rw gemax.ResponseWriter, req gemax.IncomingRequest) {
			_, _ = rw.Write([]byte(req.URL().String()))
		},
	}
	var listener = newListener(t.Name())
	var ctx, cancel = context.WithCancel(context.Background())
	t.Cleanup(cancel)
	runTask(t, func() {
		var err = server.Serve(ctx, listener)
		if err != nil {
			t.Logf("test server: Serve: %v", err)
		}
	})
	return listener
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
