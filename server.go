package gemax

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/ninedraft/gemax/status"
)

// Handler describes a gemini protocol handler.
type Handler func(ctx context.Context, rw ResponseWriter, req IncomingRequest)

// Server is gemini protocol server.
type Server struct {
	Handler     Handler
	ConnContext func(ctx context.Context, conn net.Conn) context.Context
	Logf        func(format string, args ...interface{})

	mu        sync.RWMutex
	conns     map[*connTrack]struct{}
	listeners map[net.Listener]struct{}
}

// Serve starts server on provided listener. Provided context will be passed to handlers.
func (server *Server) Serve(ctx context.Context, listener net.Listener) error {
	if server.conns == nil {
		server.conns = map[*connTrack]struct{}{}
	}
	if server.listeners == nil {
		server.listeners = map[net.Listener]struct{}{}
	}
	server.addListener(listener)
	var wg = &sync.WaitGroup{}
	defer wg.Wait()
	for {
		var conn, errAccept = listener.Accept()
		if errAccept != nil {
			return fmt.Errorf("gemini server: %w", errAccept)
		}
		wg.Add(1)
		var track = server.addConn(conn)
		go func() {
			defer wg.Done()
			defer server.removeTrack(track)
			server.handle(ctx, conn)
		}()
	}
}

// Stop gracefully shuts down the server: closes all connections.
func (server *Server) Stop() {
	server.closeAll()
}

func (server *Server) closeAll() {
	server.mu.RLock()
	defer server.mu.RUnlock()
	for conn := range server.conns {
		_ = conn.c.Close()
	}
	for listener := range server.listeners {
		_ = listener.Close()
	}
}

func (server *Server) handle(ctx context.Context, conn net.Conn) {
	defer ignoreErr(conn.Close)
	if server.ConnContext != nil {
		ctx = server.ConnContext(ctx, conn)
	}
	var deadline, deadlineOK = ctx.Deadline()
	if deadlineOK {
		_ = conn.SetDeadline(deadline)
	}
	var rw = newResponseWriter(conn)
	var req, errParseReq = ParseIncomingRequest(conn, conn.RemoteAddr().String())
	if errParseReq != nil {
		server.logf("WARN: bad request: remote_ip=%s", conn.RemoteAddr())
		rw.WriteStatus(status.PermanentFailure, "bad request")
		return
	}
	defer func() {
		if !rw.isClosed {
			_ = rw.Close()
		}
	}()
	server.Handler(ctx, rw, req)
}

func ignoreErr(fn func() error) {
	_ = fn()
}

func (server *Server) addConn(conn net.Conn) *connTrack {
	server.mu.Lock()
	defer server.mu.Unlock()
	var track = &connTrack{c: conn}
	server.conns[track] = struct{}{}
	return track
}

func (server *Server) addListener(listener net.Listener) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.listeners[listener] = struct{}{}
}

func (server *Server) removeTrack(track *connTrack) {
	server.mu.Lock()
	defer server.mu.Unlock()
	delete(server.conns, track)
}

type connTrack struct {
	c net.Conn
}

func (server *Server) logf(format string, args ...interface{}) {
	if server.Logf != nil {
		server.Logf(format, args...)
	}
}
