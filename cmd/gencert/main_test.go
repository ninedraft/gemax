package main

import (
	"crypto/x509"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestParseExpirationDate(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		got, err := parseExpirationDate("2030-01-01")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC)
		if !got.Equal(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		if _, err := parseExpirationDate("01-01-2030"); err == nil {
			t.Fatal("expected an error for invalid date format")
		}
	})
}

func TestDefaultExpiration(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 8, 10, 20, 30, 0, time.UTC)
	got := defaultExpiration(now)
	want := now.AddDate(32, 0, 0)
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNewCertificateTemplate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 8, 10, 20, 30, 0, time.UTC)
	expiration := time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC)
	serialNumber := big.NewInt(42)
	dnsNames := []string{"localhost", "example.com"}

	got := newCertificateTemplate(
		now,
		expiration,
		serialNumber,
		"dev",
		"OO",
		"ether",
		dnsNames,
	)

	if !got.NotBefore.Equal(now) {
		t.Fatalf("unexpected NotBefore: got %v, want %v", got.NotBefore, now)
	}
	if !got.NotAfter.Equal(expiration) {
		t.Fatalf("unexpected NotAfter: got %v, want %v", got.NotAfter, expiration)
	}
	if got.SerialNumber.Cmp(serialNumber) != 0 {
		t.Fatalf("unexpected serial number: got %v, want %v", got.SerialNumber, serialNumber)
	}
	if !reflect.DeepEqual(got.DNSNames, dnsNames) {
		t.Fatalf("unexpected DNS names: got %v, want %v", got.DNSNames, dnsNames)
	}
	if got.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		t.Fatal("key usage must include digital signature")
	}
	if got.KeyUsage&x509.KeyUsageKeyEncipherment == 0 {
		t.Fatal("key usage must include key encipherment")
	}
	if got.KeyUsage&x509.KeyUsageCertSign != 0 {
		t.Fatal("key usage must not include certificate signing")
	}
}

func TestValidateExpiration(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 8, 10, 20, 30, 0, time.UTC)

	t.Run("future", func(t *testing.T) {
		t.Parallel()

		if err := validateExpiration(now.Add(time.Second), now); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("equal now", func(t *testing.T) {
		t.Parallel()

		if err := validateExpiration(now, now); err == nil {
			t.Fatal("expected error for non-future expiration")
		}
	})

	t.Run("past", func(t *testing.T) {
		t.Parallel()

		if err := validateExpiration(now.Add(-time.Second), now); err == nil {
			t.Fatal("expected error for non-future expiration")
		}
	})
}

func TestGenerateSerialNumber(t *testing.T) {
	t.Parallel()

	serialNumber, err := generateSerialNumber()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if serialNumber.Sign() <= 0 {
		t.Fatalf("serial must be positive, got %v", serialNumber)
	}
	if serialNumber.BitLen() > serialNumberBits {
		t.Fatalf("serial bit length must be <= %d, got %d", serialNumberBits, serialNumber.BitLen())
	}
}

func TestWritePEMPermissions(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "key.pem")
	certPath := filepath.Join(t.TempDir(), "cert.pem")

	if err := writePEM(keyPath, "RSA PRIVATE KEY", []byte("key"), privateKeyPerm); err != nil {
		t.Fatalf("writing key: %v", err)
	}
	if err := writePEM(certPath, "CERTIFICATE", []byte("cert"), certificatePerm); err != nil {
		t.Fatalf("writing cert: %v", err)
	}

	keyInfo, errKeyInfo := os.Stat(keyPath)
	if errKeyInfo != nil {
		t.Fatalf("stat key file: %v", errKeyInfo)
	}
	if got, want := keyInfo.Mode()&os.ModePerm, fsMode(privateKeyPerm); got != want {
		t.Fatalf("key mode got %o, want %o", got, want)
	}

	certInfo, errCertInfo := os.Stat(certPath)
	if errCertInfo != nil {
		t.Fatalf("stat cert file: %v", errCertInfo)
	}
	if got, want := certInfo.Mode()&os.ModePerm, fsMode(certificatePerm); got != want {
		t.Fatalf("cert mode got %o, want %o", got, want)
	}
}

func fsMode(mode int) os.FileMode {
	return os.FileMode(mode) & os.ModePerm
}
