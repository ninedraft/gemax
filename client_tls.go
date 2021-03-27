package gemax

import (
	"crypto/tls"
	"errors"
)

var ErrInvalidServerName = errors.New("server domain and server TLS domain name don't match")

func tlsVerifyDomain(cs *tls.ConnectionState, domain string) error {
	for _, cert := range cs.PeerCertificates {
		for _, name := range cert.DNSNames {
			if name == domain {
				return nil
			}
		}
	}
	return ErrInvalidServerName
}
