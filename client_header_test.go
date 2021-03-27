package gemax_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/ninedraft/gemax"
	"github.com/ninedraft/gemax/status"
)

func TestParseResponseHeader(test *testing.T) {
	test.Run("valid header parsed successfully", func(test *testing.T) {
		var header = strings.NewReader("20 text/gemini\r\n")
		var code, meta, err = gemax.ParseResponseHeader(header)
		if err != nil {
			test.Errorf("unexpected error: %v", err)
		}
		if code != status.Success {
			test.Errorf("expected a %s, got %s", status.Text(status.Success), status.Text(code))
		}
		if meta != gemax.MIMEGemtext {
			test.Errorf("expected text/gemini, got %q", meta)
		}
	})

	test.Run("parser reads only header bytes", func(test *testing.T) {
		var response = strings.NewReader("20 text/gemini\r\n# Hello, world")
		var code, meta, err = gemax.ParseResponseHeader(response)
		if err != nil {
			test.Errorf("unexpected error: %v", err)
		}
		if code != status.Success {
			test.Errorf("expected a %s, got %s", status.Text(status.Success), status.Text(code))
		}
		if meta != gemax.MIMEGemtext {
			test.Errorf("expected text/gemini, got %q", meta)
		}
		var tail, _ = io.ReadAll(response)
		if string(tail) != "# Hello, world" {
			test.Errorf("request body must not be read by the header parser!")
		}
	})

	test.Run("parser emits error for too large responses", func(test *testing.T) {
		var header = strings.NewReader("20 text/gemini" + strings.Repeat("a", 2024) + "\r\n")
		var _, _, err = gemax.ParseResponseHeader(header)
		if !errors.Is(err, gemax.ErrHeaderTooLarge) {
			test.Errorf("unexpected error: %v, %v is expected", err, gemax.ErrHeaderTooLarge)
		}
	})

	test.Run("parser emits error for too short responses", func(test *testing.T) {
		var header = strings.NewReader("2\r\n")
		var _, _, err = gemax.ParseResponseHeader(header)
		if !errors.Is(err, gemax.ErrInvalidResponse) {
			test.Errorf("unexpected error: %v, %v is expected", err, gemax.ErrHeaderTooLarge)
		}
	})

	test.Run("parser emits error for bad status codes", func(test *testing.T) {
		var header = strings.NewReader("yadayada\r\n")
		var _, _, err = gemax.ParseResponseHeader(header)
		if !errors.Is(err, gemax.ErrInvalidResponse) {
			test.Errorf("unexpected error: %v, %v is expected", err, gemax.ErrHeaderTooLarge)
		}
	})

	test.Run("parser emits error for headers without CRLF", func(test *testing.T) {
		var header = strings.NewReader("20 text/gemini")
		var _, _, err = gemax.ParseResponseHeader(header)
		if !errors.Is(err, gemax.ErrInvalidResponse) {
			test.Errorf("unexpected error: %v, %v is expected", err, gemax.ErrHeaderTooLarge)
		}
	})
}
