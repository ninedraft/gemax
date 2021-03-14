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

	mu    sync.RWMutex
	conns map[net.Conn]struct{}
}

// Serve starts server on provided listener. Provided context will be passed to handlers.
func (server *Server) Serve(ctx context.Context, listener net.Listener) error {
	var wg = &sync.WaitGroup{}
	defer wg.Wait()
	for {
		var conn, errAccept = listener.Accept()
		if errAccept != nil {
			return fmt.Errorf("gemini server: %w", errAccept)
		}
		wg.Add(1)
		server.addConn(conn)
		go func() {
			defer wg.Done()
			defer server.removeConn(conn)
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
		_ = conn.Close()
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

	var rw = &responseWriter{dst: conn}
	var req, errParseReq = ParseIncomingRequest(conn, conn.RemoteAddr().String())
	if errParseReq != nil {
		rw.WriteStatus(status.PermanentFailure, "bad request")
		return
	}
	server.Handler(ctx, rw, req)
}

func ignoreErr(fn func() error) {
	_ = fn()
}

func (server *Server) addConn(conn net.Conn) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.conns[conn] = struct{}{}
}

func (server *Server) removeConn(conn net.Conn) {
	server.mu.Lock()
	defer server.mu.Unlock()
	delete(server.conns, conn)
}
