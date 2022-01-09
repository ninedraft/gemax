package gemax_test

import (
	"context"
	"crypto/tls"
	"embed"
	"io"
	"net"
	"testing"

	"github.com/ninedraft/gemax/gemax"
	"github.com/ninedraft/gemax/gemax/internal/testaddr"
	"github.com/ninedraft/gemax/gemax/internal/tester"
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

func TestClientTLS(test *testing.T) {
	var addr = testaddr.Addr()
	var tcpListener, errListenTCP = net.Listen("tcp", addr)
	if errListenTCP != nil {
		test.Fatalf("starting a TCP listener: %v", errListenTCP)
	}
	defer tcpListener.Close()

	var cert, errCert = tls.LoadX509KeyPair("testdata/cert.pem", "testdata/key.pem")
	if errCert != nil {
		test.Fatalf("loading test TLS certs: %v", errCert)
	}
	var tlsCfg = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var listener = tls.NewListener(tcpListener, tlsCfg)
	go func() {
		<-ctx.Done()
		listener.Close()
	}()
	go func() {
		var conn, errAccept = listener.Accept()
		if errAccept != nil {
			test.Log("accepting test connection:", errAccept)
			return
		}
		defer conn.Close()

		var testdata, errTestData = testClientPages.ReadFile("testdata/client/pages/success.com")
		if errTestData != nil {
			test.Log("reading test data:", errTestData)
			return
		}
		conn.Write(testdata)
	}()

	var client = &gemax.Client{}
	var resp, errFetch = client.Fetch(ctx, "gemini://"+addr)
	if errFetch != nil {
		test.Fatal("fetching test data:", errFetch)
	}
	defer resp.Close()

	var responseText, _ = io.ReadAll(resp)
	test.Logf("response: %q", responseText)
}
