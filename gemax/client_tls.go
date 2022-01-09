package gemax

import (
	"crypto/tls"
	"errors"
)

// ErrInvalidServerName means that the server certificate doesn't match the server domain.
var ErrInvalidServerName = errors.New("server domain and server TLS domain name don't match")

func tlsVerifyDomain(cs *tls.ConnectionState, domain string) (err error) {
	for _, cert := range cs.PeerCertificates {
		// Workaround for "x509: certificate relies on legacy Common Name field, use SANs"
		//
		// Usually self-signed certs
		if cert.Subject.CommonName == domain {
			return nil
		}
		err = cert.VerifyHostname(domain)
		if err == nil {
			return nil
		}
	}
	return ErrInvalidServerName
}
