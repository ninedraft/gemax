package gemax

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"strings"

	"golang.org/x/exp/slices"
)

var requestSuffix = []byte("\n")

// MaxRequestSize is the maximum incoming request size in bytes.
const MaxRequestSize = int64(1024 + len("\r\n"))

// IncomingRequest describes a server side request object.
type IncomingRequest interface {
	URL() *url.URL
	RemoteAddr() string
}

var (
	errDotPath = errors.New("dots in path are not permitted")
)

var ErrBadRequest = errors.New("bad request")

// ParseIncomingRequest constructs an IncomingRequest from bytestream
// and additional parameters (remote address for now).
func ParseIncomingRequest(re io.Reader, remoteAddr string) (IncomingRequest, error) {
	var certs []*x509.Certificate
	if tlsConn, ok := re.(*tls.Conn); ok {
		certs = slices.Clone(tlsConn.ConnectionState().PeerCertificates)
	}

	re = io.LimitReader(re, MaxRequestSize)

	line, errLine := io.ReadAll(re)
	if errLine != nil {
		return nil, fmt.Errorf("%w: %w", ErrBadRequest, errLine)
	}

	if !bytes.HasSuffix(line, requestSuffix) {
		return nil, ErrBadRequest
	}

	line = bytes.TrimRight(line, "\r\n")

	parsed, errParse := url.ParseRequestURI(string(line))
	if errParse != nil {
		return nil, fmt.Errorf("%w: %w", ErrBadRequest, errParse)
	}

	if !isValidPath(parsed.Path) {
		return nil, fmt.Errorf("%w: %w", ErrBadRequest, errDotPath)
	}

	if parsed.Path == "" {
		parsed.Path = "/"
	}

	return &incomingRequest{
		url:        parsed,
		remoteAddr: remoteAddr,
		certs:      certs,
	}, nil
}

func isValidPath(path string) bool {
	if path == "." {
		return false
	}

	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		return true
	}

	return fs.ValidPath(path)
}

type incomingRequest struct {
	url        *url.URL
	remoteAddr string
	certs      []*x509.Certificate
}

func (req *incomingRequest) URL() *url.URL {
	return req.url
}

func (req *incomingRequest) RemoteAddr() string {
	return req.remoteAddr
}
