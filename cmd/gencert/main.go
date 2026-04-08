package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"math/big"
	"os"
	"strings"
	"time"
)

const (
	keySize                = 4096
	dateFormat             = "2006-01-02"
	defaultExpirationYears = 32
	privateKeyPerm         = 0o600
	certificatePerm        = 0o644
	serialNumberBits       = 128
)

func main() {
	now := time.Now()

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

	expiration := defaultExpiration(now)
	flag.Func("exp",
		"certificate expiration date. Format: "+dateFormat+". Default: "+expiration.Format(dateFormat),
		func(value string) error {
			t, errParse := parseExpirationDate(value)
			if errParse != nil {
				return errParse
			}
			expiration = t
			return nil
		})
	flag.Parse()

	if errValidate := validateExpiration(expiration, now); errValidate != nil {
		log.Fatal(errValidate)
	}

	log.Print("generating private key")
	privKey, errKey := rsa.GenerateKey(rand.Reader, keySize)
	if errKey != nil {
		panic(errKey)
	}

	serialNumber, errSerial := generateSerialNumber()
	if errSerial != nil {
		panic(errSerial)
	}

	certTemplate := newCertificateTemplate(
		now,
		expiration,
		serialNumber,
		organization,
		country,
		locality,
		dnsNames,
	)

	log.Print("generating certificate")
	certEncoded, errCert := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, &privKey.PublicKey, privKey)
	if errCert != nil {
		panic(errCert)
	}
	keyEncoded := x509.MarshalPKCS1PrivateKey(privKey)

	log.Print("writing key and certificate data")
	if err := writePEM(keyOut, "RSA PRIVATE KEY", keyEncoded, privateKeyPerm); err != nil {
		panic(err)
	}
	if err := writePEM(certOut, "CERTIFICATE", certEncoded, certificatePerm); err != nil {
		panic(err)
	}
}

func writePEM(file, name string, data []byte, perm fs.FileMode) error {
	// #nosec G304 // hardcoded in code
	f, errCreate := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
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

func parseExpirationDate(value string) (time.Time, error) {
	return time.Parse(dateFormat, value)
}

func validateExpiration(expiration, now time.Time) error {
	if !expiration.After(now) {
		return fmt.Errorf("invalid -exp value %q: date must be in the future", expiration.Format(dateFormat))
	}
	return nil
}

func defaultExpiration(now time.Time) time.Time {
	return now.AddDate(defaultExpirationYears, 0, 0)
}

func generateSerialNumber() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), serialNumberBits)
	max := new(big.Int).Sub(limit, big.NewInt(1))

	serialNumber, errGenerate := rand.Int(rand.Reader, max)
	if errGenerate != nil {
		return nil, fmt.Errorf("generating serial number: %w", errGenerate)
	}

	return serialNumber.Add(serialNumber, big.NewInt(1)), nil
}

func newCertificateTemplate(
	now, expiration time.Time,
	serialNumber *big.Int,
	organization, country, locality string,
	dnsNames []string,
) *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{organization},
			Country:      []string{country},
			Locality:     []string{locality},
		},
		DNSNames:              dnsNames,
		NotBefore:             now,
		NotAfter:              expiration,
		IsCA:                  false,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}
}
