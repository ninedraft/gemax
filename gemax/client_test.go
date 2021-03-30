package gemax_test

import (
	"context"
	"embed"
	"io"
	"testing"

	"github.com/ninedraft/gemax/gemax"
	"github.com/ninedraft/gemax/gemax/internal/tester"
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
