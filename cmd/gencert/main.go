package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"
)

const (
	keySize    = 4096
	dateFormat = "2006-01-02"
)

func main() {
	keyOut := "key.pem"
	flag.StringVar(&keyOut, "key", keyOut, "dst file to write private key")

	certOut := "cert.pem"
	flag.StringVar(&certOut, "cert.pem", certOut, "dst file to write certificate")

	var dnsNames []string
	flag.Func("dns", "DNS records for cert", func(name string) error {
		if strings.TrimSpace(name) == "" {
			return nil
		}
		dnsNames = append(dnsNames, name)
		return nil
	})

	organization := "dev"
	flag.StringVar(&organization, "org", organization, "organization which generates the certificate")

	country := "OO"
	flag.StringVar(&country, "country", country, "country of certificate emitter")

	locality := "ether"
	flag.StringVar(&locality, "loc", locality, "locality of certificate emitter")

	expiration := time.Now().AddDate(32, 0, 0)
	flag.Func("exp",
		"certificate expiration date. Format: "+dateFormat+". Default: "+expiration.Format(dateFormat),
		func(value string) error {
			t, errParse := time.Parse(value, dateFormat)
			if errParse != nil {
				return errParse
			}
			expiration = t
			return nil
		})
	flag.Parse()

	log.Print("generating private key")
	privKey, errKey := rsa.GenerateKey(rand.Reader, keySize)
	if errKey != nil {
		panic(errKey)
	}

	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization: []string{organization},
			Country:      []string{country},
			Locality:     []string{locality},
		},
		DNSNames:              dnsNames,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  false,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	log.Print("generating certificate")
	certEncoded, errCert := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, &privKey.PublicKey, privKey)
	if errCert != nil {
		panic(errCert)
	}
	keyEncoded := x509.MarshalPKCS1PrivateKey(privKey)

	log.Print("writing key and certificate data")
	if err := writePEM(keyOut, "RSA PRIVATE KEY", keyEncoded); err != nil {
		panic(err)
	}
	if err := writePEM(certOut, "CERTIFICATE", certEncoded); err != nil {
		panic(err)
	}
}

func writePEM(file, name string, data []byte) error {
	// #nosec G304 // hardcoded in code
	f, errCreate := os.Create(file)
	if errCreate != nil {
		return fmt.Errorf("creating file: %w", errCreate)
	}
	defer func() { _ = f.Close() }()

	errEncode := pem.Encode(f, &pem.Block{
		Type:  name,
		Bytes: data,
	})
	if errEncode != nil {
		return fmt.Errorf("encoding PEM data: %w", errEncode)
	}
	return nil
}
