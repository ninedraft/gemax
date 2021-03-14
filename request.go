package gemax

import (
	"fmt"
	"io"
	"net/url"
)

// MaxRequestSize is the maximum incoming request size in bytes.
const MaxRequestSize = 1026

// IncomingRequest describes a server side request object.
type IncomingRequest interface {
	URL() *url.URL
	RemoteAddr() string
}

// ParseIncomingRequest constructs an IncomingRequest from bytestream
// and additional parameters (remote address for now).
func ParseIncomingRequest(re io.Reader, remoteAddr string) (IncomingRequest, error) {
	var reader = io.LimitReader(re, MaxRequestSize)
	var u string
	var _, errReadRequest = fmt.Fscanf(reader, "%s\r\n", &u)
	if errReadRequest != nil {
		return nil, fmt.Errorf("bad request: %w", errReadRequest)
	}
	var parsed, errParse = url.ParseRequestURI(u)
	if errParse != nil {
		return nil, fmt.Errorf("bad request: %w", errParse)
	}
	return &incomingRequest{
		url:        parsed,
		remoteAddr: remoteAddr,
	}, nil
}

type incomingRequest struct {
	url        *url.URL
	remoteAddr string
}

func (req *incomingRequest) URL() *url.URL {
	return req.url
}

func (req *incomingRequest) RemoteAddr() string {
	return req.remoteAddr
}
