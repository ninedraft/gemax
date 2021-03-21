package gemax

import (
	"context"
	"io"
	urlpkg "net/url"
	"path"
	"sort"
	"strings"

	"github.com/ninedraft/gemax/status"
)

// Redirect client to another page.
// This handler can work with relative
// If code is status.Success, then generates a small gemini document with a single redirect link.
// This mechanism can be used as redirect on other protocol pages.
//
// Examples:
//		Redirect(rw, req, "gemini://other.server.com/page", status.Redirect)
//		Redirect(rw, req, "../root/page", status.PermanentRedirect)
//		Redirect(rw, req, "https://wikipedia.org", status.Success)
func Redirect(rw ResponseWriter, req IncomingRequest, target string, code status.Code) {
	if code == status.Success {
		rw.WriteStatus(code, MIMEGemtext)
		_, _ = io.WriteString(rw, "=> "+target+" redirect\r\n")
		return
	}
	const geminiScheme = "gemini://"
	if strings.HasPrefix(target, geminiScheme) && len(target) > len(geminiScheme) {
		// skip URL parsing
		rw.WriteStatus(code, target)
		return
	}
	var url, errParse = urlpkg.Parse(target)
	if errParse != nil || url.Host != "" || url.Scheme != "" {
		rw.WriteStatus(code, target)
		return
	}
	// relative path
	var oldpath = req.URL().Path
	if oldpath == "" {
		oldpath = "/"
	}
	rw.WriteStatus(status.Redirect, (&urlpkg.URL{
		Scheme:   req.URL().Scheme,
		User:     req.URL().User,
		Host:     req.URL().Host,
		Path:     path.Join(oldpath, target),
		RawQuery: req.URL().RawQuery,
	}).String())
}

// NotFound serves a not found error.
func NotFound(rw ResponseWriter, req IncomingRequest) {
	rw.WriteStatus(status.NotFound, req.URL().String()+" is not found\r\n")
}

// ServeContent creates a handler, which serves provided bytes as static page.
func ServeContent(contentType string, content []byte) Handler {
	return func(_ context.Context, rw ResponseWriter, _ IncomingRequest) {
		rw.WriteStatus(status.Success, contentType)
		_, _ = rw.Write(content)
	}
}

// Query extracts canonical gemini query values
// from url query part. Values are sorted in ascending order.
// Expected values:
//		?query&key=value => [query]
//		?a&b=&key=value => [a, b]
func Query(query urlpkg.Values) []string {
	var keys = make([]string, 0, len(query))
	for key, values := range query {
		if allStringsEmpty(values) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func allStringsEmpty(items []string) bool {
	for _, value := range items {
		if value != "" {
			return false
		}
	}
	return true
}
