package gemax_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ninedraft/gemax/gemax"
	"github.com/ninedraft/gemax/gemax/internal/testaddr"
	"github.com/ninedraft/gemax/gemax/status"

	"tailscale.com/net/memnet"
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

	var resp = dialAndWrite(test, ctx, listener, "gemini://example.com/path\r\n")

	expectResponse(test, strings.NewReader(resp), "20 text/gemini\r\ngemini://example.com/path")
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

	var resp = dialAndWrite(test, ctx, listener, "invalid URL\r\n")

	expectResponse(test, strings.NewReader(resp), "59 "+status.Text(status.BadRequest)+"\r\n")
}

func TestServerInvalidHost(test *testing.T) {
	test.Parallel()
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

	var resp = dialAndWrite(test, ctx, listener, "gemini://another.com/path\r\n")

	expectResponse(test, strings.NewReader(resp), "50 host not found\r\n")
}

func TestServerCancelListen(test *testing.T) {
	test.Parallel()
	var server = &gemax.Server{
		Addr: testaddr.Addr(),
		Logf: test.Logf,
		Handler: func(_ context.Context, rw gemax.ResponseWriter, _ gemax.IncomingRequest) {
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
	if !errors.Is(err, net.ErrClosed) {
		test.Errorf("unexpected error %v, while %q is expected", err, net.ErrClosed)
	}
}

func TestListenAndServe(test *testing.T) {
	test.Parallel()
	var server = &gemax.Server{
		Addr: "localhost:40423",
		Logf: test.Logf,
		Handler: func(_ context.Context, rw gemax.ResponseWriter, _ gemax.IncomingRequest) {
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

func TestLimitedListen(test *testing.T) {
	test.Parallel()
	var trigger = make(chan struct{})
	var counter atomic.Int64

	var server = &gemax.Server{
		Addr:           testaddr.Addr(),
		Logf:           test.Logf,
		MaxConnections: 2,
		Handler: func(_ context.Context, rw gemax.ResponseWriter, _ gemax.IncomingRequest) {
			counter.Add(1)
			<-trigger
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

	var wg = sync.WaitGroup{}
	defer wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		test.Logf("test server: listening on %q", server.Addr)
		var err = server.ListenAndServe(ctx, cfg)
		switch {
		case err == nil, errors.Is(err, net.ErrClosed):
			return
		default:
			test.Errorf("test server: listening: %v", err)
		}
	}()
	time.Sleep(time.Second)

	var client = &gemax.Client{}

	wg.Add(2 * server.MaxConnections)
	for i := 0; i < 2*server.MaxConnections; i++ {
		go func() {
			defer wg.Done()
			var resp, errFetch = client.Fetch(ctx, "gemini://"+server.Addr)
			switch {
			case errFetch == nil:
				// pass
			case errors.Is(errFetch, context.Canceled):
				return
			default:
				test.Error("fetching: ", errFetch)
				return
			}
			defer func() { _ = resp.Close() }()
			expectResponse(test, resp, "example text")
			var data, errRead = io.ReadAll(resp)
			test.Logf("%s / %v", data, errRead)
		}()
	}

	time.Sleep(time.Second)
	if counter.Load() > int64(server.MaxConnections) {
		test.Errorf("number of simultaneous connections must not exceed %d", server.MaxConnections)
	}
	cancel()
	close(trigger)
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

	var resp = dialAndWrite(test, ctx, listener, "gemini://example.com/./\r\n")

	expectResponse(test, strings.NewReader(resp), "59 59 BAD REQUEST\r\n")
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

		var resp = dialAndWrite(test, ctx, listener, "gemini://example.com/notexist\r\n")

		expectResponse(test, strings.NewReader(resp), "51 gemini://example.com/notexist is not found\r\n")
	})

	test.Run("custom", func(test *testing.T) {
		test.Log("meta must not interfere with response body")
		var listener, server = setupServer(test,
			func(_ context.Context, rw gemax.ResponseWriter, _ gemax.IncomingRequest) {
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

		var resp = dialAndWrite(test, ctx, listener, "gemini://example.com/notexist\r\n")

		expectResponse(test, strings.NewReader(resp), "51 page is not found\tdotdot\r\n")
	})
}

func TestServer_Identity(test *testing.T) {
	test.Parallel()
	test.Log(
		"Check that server fetches client certificates and passes them to the handler",
	)

	called := make(chan []*x509.Certificate)
	server := gemax.Server{
		Logf:  test.Logf,
		Addr:  testaddr.Addr(),
		Hosts: []string{"example.com"},
		Handler: func(_ context.Context, rw gemax.ResponseWriter, req gemax.IncomingRequest) {
			rw.WriteStatus(status.Success, "example text")
			called <- req.Certificates()
		},
	}

	ctx := context.Background()

	go func() {
		_ = server.ListenAndServe(ctx, &tls.Config{
			//nolint:gosec // G402 - it's ok to skip verification for gemini server
			InsecureSkipVerify: true,
			Certificates:       []tls.Certificate{serverCert},
			ClientAuth:         tls.RequireAnyClientCert,
		})
	}()

	cfg := &tls.Config{
		ServerName: "example.com",
		//nolint:gosec // G402 - it's ok to skip verification for gemini server
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
		Certificates:       []tls.Certificate{clientCert},
		VerifyConnection:   func(tls.ConnectionState) error { return nil },
		VerifyPeerCertificate: func([][]byte, [][]*x509.Certificate) error {
			return nil
		},
	}

	var conn net.Conn

	// wait for server to start
	for i := 0; true; i++ {
		c, errDial := tls.Dial("tcp", server.Addr, cfg)

		switch {
		case i >= 20 && errDial != nil:
			test.Fatal("server is not started: %w", errDial)
		case errDial != nil:
			test.Log("server is not started yet, retrying...")
			continue
		}
		conn = c
		break
	}

	_, _ = fmt.Fprintf(conn, "gemini://example.com/\r\n")
	_ = conn.Close()

	gotCerts := <-called
	if len(gotCerts) != 1 {
		test.Fatalf("got %d certificates, want 1", len(gotCerts))
	}
	assertEq(test, gotCerts[0].Subject.CommonName, "client", "certificate CN")
}

func setupServer(t *testing.T, handler gemax.Handler) (*memnet.Listener, *gemax.Server) {
	t.Helper()
	var server = &gemax.Server{
		Logf:    t.Logf,
		Handler: handler,
	}
	var listener = memnet.Listen(t.Name())
	return listener, server
}

func setupEchoServer(t *testing.T) (*memnet.Listener, *gemax.Server) {
	t.Helper()
	return setupServer(t, func(_ context.Context, rw gemax.ResponseWriter, req gemax.IncomingRequest) {
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

//nolint:unparam // it's ok for tests
func dialAndWrite(t *testing.T, ctx context.Context, dialer *memnet.Listener, format string, args ...any) string {
	t.Helper()

	t.Log("dialing in-memory network")
	conn, errDial := dialer.Dial(ctx, "tcp", t.Name())
	if errDial != nil {
		panic("dialin in-memory network: " + errDial.Error())
	}

	defer func() { _ = conn.Close() }()

	t.Log("writing to in-memory network")
	_, errWrite := fmt.Fprintf(conn, format, args...)
	if errWrite != nil {
		panic("writing to in-memory network: " + errWrite.Error())
	}

	var resp = &strings.Builder{}

	t.Log("reading from in-memory network")
	_, errRead := io.Copy(resp, conn)
	if errRead != nil {
		panic("reading from in-memory network: " + errRead.Error())
	}

	return resp.String()
}
