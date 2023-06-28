package router_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/ninedraft/gemax/gemax"
	"github.com/ninedraft/gemax/gemax/internal/tester"
	"github.com/ninedraft/gemax/gemax/router"
)

func TestRouter_HandleParams(test *testing.T) {
	test.Parallel()

	r := router.New()
	ctx := context.Background()

	r.HandleParams("/hello/world",
		func(_ context.Context, rw gemax.ResponseWriter, req router.IncomingRequest) {
			assertEqual(test, req.URL().Path, "/hello/world")

			param, ok := req.Param("name")
			assertEqual(test, param, "")
			assertEqual(test, ok, false)

			_, _ = io.WriteString(rw, req.URL().Path)
		})

	r.HandleParams("/hello",
		func(_ context.Context, rw gemax.ResponseWriter, req router.IncomingRequest) {
			assertEqual(test, req.URL().Path, "/hello")

			param, ok := req.Param("name")
			assertEqual(test, param, "")
			assertEqual(test, ok, false)

			_, _ = io.WriteString(rw, req.URL().Path)
		})

	r.HandleParams("/hello/:name",
		func(_ context.Context, rw gemax.ResponseWriter, req router.IncomingRequest) {
			if strings.Contains(req.URL().Path, ":") {
				test.Errorf("expected no params in path, got %s", req.URL().Path)
			}

			param, ok := req.Param("name")
			assertNotEqual(test, "", param)
			assertEqual(test, ok, true)

			_, _ = io.WriteString(rw, req.URL().Path)
		})

	test.Run("match-no-params", func(t *testing.T) {
		t.Parallel()

		response := &tester.ResponseWriter{}
		req := tester.NewIncomingRequest("/hello/world", "")

		serverd := r.Serve(ctx, response, req)

		assertEqual(t, response.Body.String(), "/hello/world")
		assertEqual(t, serverd, true)
	})

	test.Run("match-with-params", func(t *testing.T) {
		t.Parallel()

		response := &tester.ResponseWriter{}
		req := tester.NewIncomingRequest("/hello/world", "")

		serverd := r.Serve(ctx, response, req)

		assertEqual(t, "/hello/world", response.Body.String())
		assertEqual(t, serverd, true)
	})

	test.Run("match-short", func(t *testing.T) {
		t.Parallel()

		response := &tester.ResponseWriter{}
		req := tester.NewIncomingRequest("/hello/merlin", "")

		serverd := r.Serve(ctx, response, req)

		assertEqual(t, "/hello/merlin", response.Body.String())
		assertEqual(t, serverd, true)
	})

	test.Run("no-match", func(t *testing.T) {
		t.Parallel()

		response := &tester.ResponseWriter{}
		req := tester.NewIncomingRequest("/fasan", "")

		served := r.Serve(ctx, response, req)

		assertEqual(t, response.Body.String(), "")
		assertEqual(t, false, served)
	})
}

func assertEqual[E comparable](t *testing.T, a, b E) {
	t.Helper()

	if a != b {
		t.Errorf("Expected %v, got %v", a, b)
	}
}

func assertNotEqual[E comparable](t *testing.T, a, b E) {
	t.Helper()

	if a == b {
		t.Errorf("Expected not %v, got %v", a, b)
	}
}
