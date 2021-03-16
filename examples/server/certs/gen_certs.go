package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

func main() {
	var key, errPrivateKey = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if errPrivateKey != nil {
		panic(errPrivateKey)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"gemax"},
		},
		NotBefore:          time.Now(),
		NotAfter:           time.Now().Add(10 * 365 * 24 * time.Hour),
		SignatureAlgorithm: x509.ECDSAWithSHA512,
		PublicKeyAlgorithm: x509.RSA,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,

		DNSNames: []string{"localhost"},
	}

	var derBytes, errCert = x509.CreateCertificate(rand.Reader, &template, &template, publicKey(key), key)
	if errCert != nil {
		var msg = fmt.Sprintf("Failed to create certificate: %s", errCert)
		panic(msg)
	}
	out := &bytes.Buffer{}
	pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	os.WriteFile("cert.pem", out.Bytes(), 0644)

	out.Reset()
	pem.Encode(out, pemBlockForKey(key))
	os.WriteFile("key.pem", out.Bytes(), 0644)
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}
