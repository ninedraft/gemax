package gemax

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	urlpkg "net/url"
	"runtime"
	"strings"
	"sync"

	"github.com/ninedraft/gemax/gemax/internal/bufreader"
	"github.com/ninedraft/gemax/gemax/status"
)

// Client is used to fetch gemini resources.
// Empty client value cane be considered as initialized.
type Client struct {
	MaxResponseSize int64
	Dial            func(ctx context.Context, host string, cfg *tls.Config) (net.Conn, error)
	// CheckRedirect specifies the policy for handling redirects.
	// If CheckRedirect is not nil, the client calls it before
	// following an Gemini redirect. The arguments req and via are
	// the upcoming request and the requests made already, oldest
	// first. If CheckRedirect returns an error, the Client's Fetch
	// method returns both the previous Response (with its Body
	// closed) and CheckRedirect's error.
	// instead of issuing the Request req.
	// As a special case, if CheckRedirect returns ErrUseLastResponse,
	// then the most recent response is returned with its body
	// unclosed, along with a nil error.
	//
	// If CheckRedirect is nil, the Client uses its default policy,
	// which is to stop after 10 consecutive requests.
	CheckRedirect func(ctx context.Context, verification *urlpkg.URL, via []RedirectedRequest) error
	once          sync.Once
}

var (
	// ErrTooManyRedirects means that server tried through too many adresses.
	// Default limit is 10.
	// User implementations of CheckRedirect should use this error then limiting number of redirects.
	ErrTooManyRedirects = errors.New("too many redirects")
)

func (client *Client) checkRedirect(ctx context.Context, req *urlpkg.URL, via []RedirectedRequest) error {
	if client.CheckRedirect != nil {
		return client.CheckRedirect(ctx, req, via)
	}
	return defaultRedirect(ctx, req, via)
}

func defaultRedirect(_ context.Context, _ *urlpkg.URL, via []RedirectedRequest) error {
	const max = 10
	if len(via) < max {
		return nil
	}
	return ErrTooManyRedirects
}

// RedirectedRequest  contains executed gemini request data
// and corresponding response with closed body.
type RedirectedRequest struct {
	Req      *urlpkg.URL
	Response *Response
}

const readerBufSize = 16 << 10

// Fetch gemini resource.
func (client *Client) Fetch(ctx context.Context, url string) (*Response, error) {
	client.init()
	//nolint:prealloc // unable to preallocate, we don't know number of redirects
	var redirects []RedirectedRequest
	for {
		var u, errParseURL = urlpkg.Parse(url)
		if errParseURL != nil {
			return nil, fmt.Errorf("parsing URL: %w", errParseURL)
		}
		if err := client.checkRedirect(ctx, u, redirects); err != nil {
			return nil, fmt.Errorf("redirect: %w", err)
		}
		resp, errFetch := client.fetch(ctx, url, u)
		if errFetch != nil {
			return resp, errFetch
		}
		if !isRedirect(resp.Status) {
			return resp, nil
		}
		_ = resp.Close()
		redirects = append(redirects, RedirectedRequest{
			Req:      u,
			Response: resp,
		})
		url = resp.Meta
	}
}

func isRedirect(code status.Code) bool {
	return code == status.Redirect || code == status.RedirectPermanent
}

func (client *Client) fetch(ctx context.Context, origURL string, u *urlpkg.URL) (*Response, error) {
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

	var _, errWrite = io.WriteString(conn, origURL+"\r\n")
	if errWrite != nil {
		return nil, fmt.Errorf("sending request: %w", errWrite)
	}

	var re = bufreader.New(conn, readerBufSize)
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
		Config:    cfg,
	}
	return tlsDialer.DialContext(ctx, "tcp", host)
}

func (client *Client) init() {
	client.once.Do(func() {})
}

// Response contains parsed server response.
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
