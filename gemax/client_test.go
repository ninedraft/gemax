package gemax_test

import (
	"context"
	"embed"
	"errors"
	"io"
	"testing"

	"github.com/ninedraft/gemax/gemax"
	"github.com/ninedraft/gemax/gemax/internal/tester"
	"github.com/ninedraft/gemax/gemax/status"
)

//go:embed testdata/client/pages/*
var testClientPages embed.FS

func TestClient(test *testing.T) {
	var dialer = tester.DialFS{
		Prefix: "testdata/client/pages/",
		FS:     testClientPages,
	}
	var client = &gemax.Client{
		Dial: dialer.Dial,
	}
	var ctx = context.Background()
	var resp, errFetch = client.Fetch(ctx, "gemini://success.com")
	if errFetch != nil {
		test.Errorf("unexpected fetch error: %v", errFetch)
		return
	}
	defer func() { _ = resp.Close() }()
	var data, errRead = io.ReadAll(resp)
	if errRead != nil {
		test.Errorf("unexpected error while reading response body: %v", errRead)
		return
	}
	test.Logf("%s", data)
}

func TestClient_Redirect(test *testing.T) {
	var dialer = tester.DialFS{
		Prefix: "testdata/client/pages/",
		FS:     testClientPages,
	}
	var client = &gemax.Client{
		Dial: dialer.Dial,
	}
	var ctx = context.Background()
	var resp, errFetch = client.Fetch(ctx, "gemini://redirect1.com")
	if errFetch != nil {
		test.Errorf("unexpected fetch error: %v", errFetch)
		return
	}
	if resp.Status != status.Success {
		test.Fatalf("unexpected status code %v", resp.Status)
	}
	defer func() { _ = resp.Close() }()
	var data, errRead = io.ReadAll(resp)
	if errRead != nil {
		test.Errorf("unexpected error while reading response body: %v", errRead)
		return
	}
	test.Logf("%s", data)
}

func TestClient_InfiniteRedirect(test *testing.T) {
	var dialer = tester.DialFS{
		Prefix: "testdata/client/pages/",
		FS:     testClientPages,
	}
	var client = &gemax.Client{
		Dial: dialer.Dial,
	}
	var ctx = context.Background()
	var _, errFetch = client.Fetch(ctx, "gemini://redirect2.com")
	switch {
	case errors.Is(errFetch, gemax.ErrTooManyRedirects):
		// ok
	case errFetch != nil:
		test.Fatalf("unexpected error %q", errFetch)
	default:
		test.Fatalf("an error is expected, got nil")
	}
}
