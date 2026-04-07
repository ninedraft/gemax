package gemax_test

import (
	"context"
	"crypto/tls"
	"embed"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ninedraft/gemax/gemax"
	"github.com/ninedraft/gemax/gemax/internal/tester"
	"github.com/ninedraft/gemax/gemax/status"
)

//go:embed testdata/client/pages/*
var testClientPages embed.FS

func TestClient(test *testing.T) {
	var dialer = tester.DialFS{
		Prefix: "testdata/client/pages/",
		FS:     testClientPages,
	}
	var client = &gemax.Client{
		Dial: dialer.Dial,
	}
	var ctx = context.Background()
	var resp, errFetch = client.Fetch(ctx, "gemini://success.com")
	if errFetch != nil {
		test.Errorf("unexpected fetch error: %v", errFetch)
		return
	}
	defer func() { _ = resp.Close() }()
	var data, errRead = io.ReadAll(resp)
	if errRead != nil {
		test.Errorf("unexpected error while reading response body: %v", errRead)
		return
	}
	test.Logf("%s", data)
}

func TestClient_Redirect(test *testing.T) {
	var dialer = tester.DialFS{
		Prefix: "testdata/client/pages/",
		FS:     testClientPages,
	}
	var client = &gemax.Client{
		Dial: dialer.Dial,
	}
	var ctx = context.Background()
	var resp, errFetch = client.Fetch(ctx, "gemini://redirect1.com")
	if errFetch != nil {
		test.Errorf("unexpected fetch error: %v", errFetch)
		return
	}
	if resp.Status != status.Success {
		test.Fatalf("unexpected status code %v", resp.Status)
	}
	defer func() { _ = resp.Close() }()
	var data, errRead = io.ReadAll(resp)
	if errRead != nil {
		test.Errorf("unexpected error while reading response body: %v", errRead)
		return
	}
	test.Logf("%s", data)
}

func TestClient_InfiniteRedirect(test *testing.T) {
	var dialer = tester.DialFS{
		Prefix: "testdata/client/pages/",
		FS:     testClientPages,
	}
	var client = &gemax.Client{
		Dial: dialer.Dial,
	}
	var ctx = context.Background()
	var _, errFetch = client.Fetch(ctx, "gemini://redirect2.com")
	switch {
	case errors.Is(errFetch, gemax.ErrTooManyRedirects):
		// ok
	case errFetch != nil:
		test.Fatalf("unexpected error %q", errFetch)
	default:
		test.Fatalf("an error is expected, got nil")
	}
}

func TestClient_Fetch_ClosesConnOnWriteError(test *testing.T) {
	test.Parallel()
	var errWrite = errors.New("write failed")
	var conn = &recordingConn{
		writeErr: errWrite,
		reader:   strings.NewReader(""),
	}
	var client = &gemax.Client{
		Dial: func(context.Context, string, *tls.Config) (net.Conn, error) {
			return conn, nil
		},
	}
	var _, errFetch = client.Fetch(context.Background(), "gemini://example.com")
	if !errors.Is(errFetch, errWrite) {
		test.Fatalf("unexpected fetch error: %v", errFetch)
	}
	if conn.closeCalls != 1 {
		test.Fatalf("connection must be closed once, got %d", conn.closeCalls)
	}
}

func TestClient_Fetch_ClosesConnOnHeaderParseError(test *testing.T) {
	test.Parallel()
	var conn = &recordingConn{
		reader: strings.NewReader("not-a-valid-header\r\n"),
	}
	var client = &gemax.Client{
		Dial: func(context.Context, string, *tls.Config) (net.Conn, error) {
			return conn, nil
		},
	}
	var _, errFetch = client.Fetch(context.Background(), "gemini://example.com")
	if !errors.Is(errFetch, gemax.ErrInvalidResponse) {
		test.Fatalf("unexpected fetch error: %v", errFetch)
	}
	if conn.closeCalls != 1 {
		test.Fatalf("connection must be closed once, got %d", conn.closeCalls)
	}
}

type recordingConn struct {
	reader     io.Reader
	writeErr   error
	closeCalls int
}

func (conn *recordingConn) Read(data []byte) (int, error) {
	if conn.reader == nil {
		return 0, io.EOF
	}
	return conn.reader.Read(data)
}

func (conn *recordingConn) Write(data []byte) (int, error) {
	if conn.writeErr != nil {
		return 0, conn.writeErr
	}
	return len(data), nil
}

func (conn *recordingConn) Close() error {
	conn.closeCalls++
	return nil
}

func (conn *recordingConn) LocalAddr() net.Addr {
	return &net.TCPAddr{}
}

func (conn *recordingConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{}
}

func (conn *recordingConn) SetDeadline(time.Time) error {
	return nil
}

func (conn *recordingConn) SetReadDeadline(time.Time) error {
	return nil
}

func (conn *recordingConn) SetWriteDeadline(time.Time) error {
	return nil
}
