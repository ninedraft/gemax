package gemax

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	urlpkg "net/url"
	"runtime"
	"strings"
	"sync"

	"github.com/ninedraft/gemax/internal/bufreader"
	"github.com/ninedraft/gemax/status"
)

type Client struct {
	MaxResponseSize int64
	Dial            func(ctx context.Context, host string, cfg *tls.Config) (net.Conn, error)
	once            sync.Once
}

func (client *Client) Fetch(ctx context.Context, url string) (*Response, error) {
	client.init()
	var u, errParseURL = urlpkg.Parse(url)
	if errParseURL != nil {
		return nil, fmt.Errorf("parsing URL: %w", errParseURL)
	}

	var host = u.Host
	if strings.LastIndexByte(host, ':') < 0 {
		host += ":1965"
	}
	var domain, _, _ = net.SplitHostPort(host)
	var conn, errConn = client.dial(ctx, host, &tls.Config{
		MinVersion: tls.VersionTLS12,
		//nolint:gosec // we skipping certificate verification because gemini servers usually don't use CAs
		InsecureSkipVerify: true,
		VerifyConnection: func(cs tls.ConnectionState) error {
			return tlsVerifyDomain(&cs, domain)
		},
	})
	if errConn != nil {
		return nil, fmt.Errorf("connecting to the server %q: %w", host, errConn)
	}
	ctxConnDeadline(ctx, conn)

	var _, errWrite = io.WriteString(conn, url+"\r\n")
	if errWrite != nil {
		return nil, fmt.Errorf("sending request: %w", errWrite)
	}

	var re = bufreader.New(conn, 16<<10, client.MaxResponseSize)
	var code, meta, errHeader = ParseResponseHeader(re)
	if errHeader != nil {
		return nil, errHeader
	}
	var resp = &Response{
		Status: code,
		Meta:   meta,
		reader: re,
	}
	runtime.SetFinalizer(resp, func(resp *Response) {
		_ = resp.Close()
	})
	return resp, nil
}

func (client *Client) dial(ctx context.Context, host string, cfg *tls.Config) (net.Conn, error) {
	if client.Dial != nil {
		return client.Dial(ctx, host, cfg)
	}
	var tlsDialer = &tls.Dialer{
		NetDialer: &net.Dialer{},
	}
	return tlsDialer.DialContext(ctx, "tcp", host)
}

func (client *Client) init() {
	client.once.Do(func() {})
}

type Response struct {
	Status status.Code
	Meta   string
	reader
}

type reader interface {
	io.Reader
	io.ByteReader
	io.RuneReader
	io.Closer
}

func ctxConnDeadline(ctx context.Context, conn net.Conn) {
	var deadline, ok = ctx.Deadline()
	if ok {
		_ = conn.SetDeadline(deadline)
	}
}
