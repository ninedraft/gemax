package main

import (
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
	dnsNames := []string{"localhost", "example.com"}

	got := newCertificateTemplate(
		now,
		expiration,
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
	if !reflect.DeepEqual(got.DNSNames, dnsNames) {
		t.Fatalf("unexpected DNS names: got %v, want %v", got.DNSNames, dnsNames)
	}
}
