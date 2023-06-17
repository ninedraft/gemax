package gemax_test

import (
	"strings"
	"testing"

	"github.com/ninedraft/gemax/gemax"
)

func TestParseIncomingRequest(t *testing.T) {
	t.Parallel()
	t.Log("parsing incoming request line")

	const remoteAddr = "remote-addr"
	type expect struct {
		err bool
		url string
	}

	tc := func(name, input string, expected expect) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			re := strings.NewReader(input)

			parsed, err := gemax.ParseIncomingRequest(re, remoteAddr)

			if (err != nil) != expected.err {
				t.Errorf("error = %v, want error = %v", err, expected.err)
			}

			if parsed == nil && err == nil {
				t.Error("parsed = nil, want not nil")
				return
			}

			if parsed != nil {
				assertEq(t, parsed.RemoteAddr(), remoteAddr, "remote addr")
				assertEq(t, parsed.URL().String(), expected.url, "url")
			}
		})
	}

	tc("valid",
		"gemini://example.com\r\n", expect{
			url: "gemini://example.com/",
		})
	tc("valid no \\r",
		"gemini://example.com\n", expect{
			url: "gemini://example.com/",
		})
	tc("valid with path",
		"gemini://example.com/path\r\n", expect{
			url: "gemini://example.com/path",
		})
	tc("valid with path and query",
		"gemini://example.com/path?query=value\r\n", expect{
			url: "gemini://example.com/path?query=value",
		})
	tc("valid http",
		"http://example.com\r\n", expect{
			url: "http://example.com/",
		})

	tc("too long",
		"http://example.com/"+strings.Repeat("a", 2048)+"\r\n",
		expect{err: true})
	tc("empty",
		"", expect{err: true})
	tc("no new \\r\\n",
		"gemini://example.com", expect{err: true})
	tc("no \\n",
		"gemini://example.com\r", expect{err: true})
}

func assertEq[E comparable](t *testing.T, got, want E, format string, args ...any) {
	t.Helper()

	if got != want {
		t.Errorf("got %v, want %v", got, want)
		t.Errorf(format, args...)
	}
}
