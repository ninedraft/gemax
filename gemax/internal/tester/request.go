package tester

import (
	"crypto/x509"
	"net/url"

	"github.com/ninedraft/gemax/gemax"
)

var _ gemax.IncomingRequest = &incomingRequest{}

func NewIncomingRequest(u, remoteAddr string) gemax.IncomingRequest {
	var parsed, errParse = url.Parse(u)
	if errParse != nil {
		panic("parsing url: " + errParse.Error())
	}

	return &incomingRequest{
		url:        parsed,
		remoteAddr: remoteAddr,
	}
}

type incomingRequest struct {
	url        *url.URL
	remoteAddr string
	params     map[string]string
}

func (req *incomingRequest) URL() *url.URL {
	return req.url
}

func (req *incomingRequest) RemoteAddr() string {
	return req.remoteAddr
}

func (*incomingRequest) Certificates() []*x509.Certificate {
	return nil
}

func (req *incomingRequest) Param(name string) (string, bool) {
	if req == nil || req.params == nil {
		return "", false
	}

	val, ok := req.params[name]
	return val, ok
}

func (req *incomingRequest) WithParam(name, value string) *incomingRequest {
	if req.params == nil {
		req.params = make(map[string]string)
	}

	req.params[name] = value

	return req
}
