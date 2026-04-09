package gemax_test

import (
	"bytes"
	"context"
	"crypto/x509"
	urlpkg "net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/ninedraft/gemax/gemax"
	"github.com/ninedraft/gemax/gemax/status"
)

func TestServeContent(test *testing.T) {
	var testdata = []byte("example text")
	var ctx = context.Background()
	var rw = &responseRecorder{}
	var req = &request{
		remoteAddr: test.Name(),
		url:        "gemini://localhost.example.net",
	}
	var serve = gemax.ServeContent("application/octet-stream", testdata)

	serve(ctx, rw, req)

	if rw.status != status.Success {
		test.Errorf("%s is expected, got %s", status.Success, rw.status)
	}
	if !bytes.Equal(rw.Bytes(), testdata) {
		test.Errorf("%q expected, got %q", testdata, rw)
	}
}

func TestQuery(test *testing.T) {
	var t = func(query string, expected []string) {
		var name = query + "->[" + strings.Join(expected, ",") + "]"
		test.Run(name, func(test *testing.T) {
			var parsed, errParse = urlpkg.ParseQuery(query)
			if errParse != nil {
				panic("invalid test query value: " + errParse.Error())
			}
			test.Logf("parsed query: %+q", parsed)

			var values = gemax.Query(parsed)

			if !reflect.DeepEqual(values, expected) {
				test.Errorf("expected %q, got %q", expected, values)
			}
		})
	}

	t("query&foo=bar", []string{"query"})
	t("query=&foo=bar", []string{"query"})
	t("query&foo=bar,1", []string{"query"})
	t("query&foo=bar&query", []string{"query"})
	t("foo=bar", []string{})
}

func TestRedirect(test *testing.T) {
	var runCase = func(name, reqURL, target, wantMeta string) {
		test.Run(name, func(test *testing.T) {
			var rw = &responseRecorder{}
			var req = &request{
				remoteAddr: test.Name(),
				url:        reqURL,
			}

			gemax.Redirect(rw, req, target, status.RedirectPermanent)

			if rw.status != status.RedirectPermanent {
				test.Fatalf("expected %s, got %s", status.RedirectPermanent, rw.status)
			}
			if rw.meta != wantMeta {
				test.Fatalf("expected %q, got %q", wantMeta, rw.meta)
			}
		})
	}

	runCase("relative page target", "gemini://example.com/a/page.gmi", "b.gmi", "gemini://example.com/a/b.gmi")
	runCase("relative target does not preserve source query", "gemini://example.com/a/page.gmi?old=q", "b.gmi", "gemini://example.com/a/b.gmi")
	runCase("query-only target updates query", "gemini://example.com/a/page.gmi?old=q", "?new=q", "gemini://example.com/a/page.gmi?new=q")
	runCase("absolute-path target resolves from host root", "gemini://example.com/a/page.gmi", "/root.gmi", "gemini://example.com/root.gmi")
	runCase("path without trailing slash is treated as file", "gemini://example.com/a", "b", "gemini://example.com/b")
	runCase("path with trailing slash is treated as directory", "gemini://example.com/a/", "b", "gemini://example.com/a/b")
	runCase("absolute gemini target passes through", "gemini://example.com/a/page.gmi", "gemini://other.host/x?z=1", "gemini://other.host/x?z=1")
}

type request struct {
	remoteAddr string
	url        string
}

func (req *request) URL() *urlpkg.URL {
	var u, _ = urlpkg.Parse(req.url)
	return u
}

func (req *request) RemoteAddr() string {
	return req.remoteAddr
}

func (req *request) Certificates() []*x509.Certificate {
	return nil
}

type responseRecorder struct {
	status status.Code
	meta   string
	bytes.Buffer
}

func (r *responseRecorder) Close() error {
	return nil
}

func (r *responseRecorder) WriteStatus(code status.Code, meta string) {
	if r.status != 0 {
		return
	}
	if code == status.Success && meta == "" {
		meta = gemax.MIMEGemtext
	}
	r.status = code
	r.meta = meta
}
