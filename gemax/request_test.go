package gemax_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ninedraft/gemax/gemax"

	"tailscale.com/net/memnet"
)

var (
	clientCert = testCert("client")
	serverCert = testCert("server")
)

func TestParseIncomingRequest(t *testing.T) {
	t.Parallel()
	t.Log("parsing incoming request line")

	const remoteAddr = "remote-addr"
	type expect struct {
		err bool
		url string
	}

	tc := func(name, input string, expected expect) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			re := strings.NewReader(input)

			parsed, err := gemax.ParseIncomingRequest(re, remoteAddr)

			if (err != nil) != expected.err {
				t.Errorf("error = %v, want error = %v", err, expected.err)
			}

			if parsed == nil && err == nil {
				t.Error("parsed = nil, want not nil")
				return
			}

			if parsed != nil {
				assertEq(t, parsed.RemoteAddr(), remoteAddr, "remote addr")
				assertEq(t, parsed.URL().String(), expected.url, "url")
			}
		})
	}

	tc("valid",
		"gemini://example.com\r\n", expect{
			url: "gemini://example.com/",
		})
	tc("valid no \\r",
		"gemini://example.com\n", expect{
			url: "gemini://example.com/",
		})
	tc("valid with path",
		"gemini://example.com/path\r\n", expect{
			url: "gemini://example.com/path",
		})
	tc("valid with path and query",
		"gemini://example.com/path?query=value\r\n", expect{
			url: "gemini://example.com/path?query=value",
		})
	tc("valid http",
		"http://example.com\r\n", expect{
			url: "http://example.com/",
		})

	tc("too long",
		"http://example.com/"+strings.Repeat("a", 2048)+"\r\n",
		expect{err: true})
	tc("empty",
		"", expect{err: true})
	tc("no new \\r\\n",
		"gemini://example.com", expect{err: true})
	tc("no \\n",
		"gemini://example.com\r", expect{err: true})
}

func TestRequest_Certificates(test *testing.T) {
	test.Parallel()
	test.Log("Test that we can get the client certificates from the client request")

	wg := sync.WaitGroup{}
	defer wg.Wait()

	a, b := memnet.NewConn(test.Name(), 10<<20)
	defer func() {
		_ = a.Close()
		_ = b.Close()
	}()

	deadline := time.Now().Add(5 * time.Second)
	_ = a.SetDeadline(deadline)
	_ = b.SetDeadline(deadline)

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	test.Log("setting up server")
	server := tls.Server(a, &tls.Config{
		//nolint:gosec // G402 - it's ok to skip verification for gemini server
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
		Certificates:       []tls.Certificate{serverCert},
		ClientAuth:         tls.RequireAnyClientCert,
		VerifyPeerCertificate: func([][]byte, [][]*x509.Certificate) error {
			return nil
		},
		VerifyConnection: func(tls.ConnectionState) error {
			return nil
		},
	})
	defer func() { _ = server.Close() }()

	test.Log("setting up client")
	client := tls.Client(b, &tls.Config{
		//nolint:gosec // G402 - it's ok to skip verification for gemini server
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
		Certificates:       []tls.Certificate{clientCert},
		VerifyConnection:   func(tls.ConnectionState) error { return nil },
		VerifyPeerCertificate: func([][]byte, [][]*x509.Certificate) error {
			return nil
		},
	})

	wg.Add(1)
	go func() {
		defer func() { _ = client.Close() }()
		defer wg.Done()

		test.Log("sending request")
		_, errRequest := fmt.Fprintf(client, "gemini://localhost:1968\r\n")
		if errRequest != nil {
			test.Error("sending request:", errRequest)
			return
		}
	}()

	// run handshake manually, because ParseIncomingRequest
	// accesses the connection state before the handshake is complete
	test.Log("server: handshaking client")
	if err := server.HandshakeContext(ctx); err != nil {
		test.Fatal("server handshake:", err)
	}

	test.Log("handling request")
	req, errParseReq := gemax.ParseIncomingRequest(server, test.Name())
	if errParseReq != nil {
		test.Fatal("parsing request:", errParseReq)
	}

	certs := req.Certificates()
	if len(certs) == 0 {
		test.Error("no certificates in incoming request")
		return
	}
	assertEq(test, certs[0].Issuer.CommonName, "client", "client cert issuer")
}

func assertEq[E comparable](t *testing.T, got, want E, format string, args ...any) {
	t.Helper()

	if got != want {
		t.Errorf("got %v, want %v", got, want)
		t.Errorf(format, args...)
	}
}

func testCert(organization string) tls.Certificate {
	privateKey, errGenerate := rsa.GenerateKey(rand.Reader, 2048)
	if errGenerate != nil {
		panic("failed to generate private key: " + errGenerate.Error())
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(int64(time.Now().Year())),
		Subject: pkix.Name{
			CommonName:   organization,
			Organization: []string{organization},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour), // Valid for 1 year
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	derBytes, errCert := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if errCert != nil {
		panic("failed to create certificate: " + errCert.Error())
	}

	return tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  privateKey,
	}
}
