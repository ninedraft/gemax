package gemax

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"sync"

	"github.com/ninedraft/gemax/gemax/status"
)

// Handler describes a gemini protocol handler.
type Handler func(ctx context.Context, rw ResponseWriter, req IncomingRequest)

// Server is gemini protocol server.
type Server struct {
	Addr string
	// Hosts expected by server.
	// If empty, then every host will be valid.
	Hosts       []string
	Handler     Handler
	ConnContext func(ctx context.Context, conn net.Conn) context.Context
	Logf        func(format string, args ...interface{})

	mu        sync.RWMutex
	conns     map[*connTrack]struct{}
	listeners map[net.Listener]struct{}

	once  sync.Once
	hosts map[string]struct{}
}

func (server *Server) init() {
	server.once.Do(func() {
		server.conns = map[*connTrack]struct{}{}
		server.listeners = map[net.Listener]struct{}{}
		server.buildHosts()
	})
}

// ListenAndServe starts a TLS gemini server at specified server.
func (server *Server) ListenAndServe(ctx context.Context, tlsCfg *tls.Config) error {
	server.init()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var lc = net.ListenConfig{}
	var tcpListener, errListener = lc.Listen(ctx, "tcp", server.Addr)
	if errListener != nil {
		return fmt.Errorf("creating listener: %w", errListener)
	}
	var listener = tls.NewListener(tcpListener, tlsCfg)
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()
	server.addListener(listener)
	defer ignoreErr(listener.Close)
	return server.Serve(ctx, listener)
}

// Serve starts server on provided listener. Provided context will be passed to handlers.
func (server *Server) Serve(ctx context.Context, listener net.Listener) error {
	server.init()
	server.addListener(listener)
	for {
		var conn, errAccept = listener.Accept()
		if errAccept != nil {
			return fmt.Errorf("gemini server: %w", errAccept)
		}
		var track = server.addConn(conn)
		go func() {
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
	defer func() {
		if !rw.isClosed {
			_ = rw.Close()
		}
	}()
	var req, errParseReq = ParseIncomingRequest(conn, conn.RemoteAddr().String())
	if errParseReq != nil {
		server.logf("WARN: bad request: remote_ip=%s", conn.RemoteAddr())
		const code = status.BadRequest
		rw.WriteStatus(code, status.Text(code))
		return
	}
	if !server.validHost(req.URL()) {
		server.logf("WARN: bad request: unknown host %q", req.URL().Host)
		rw.WriteStatus(status.PermanentFailure, "host not found")
		return
	}

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

func (server *Server) validHost(u *url.URL) bool {
	if u.Host == "" {
		return false
	}
	if len(server.hosts) == 0 {
		return true
	}
	var host = u.Host
	var hostname = u.Hostname()
	var _, hostOk = server.hosts[host]
	var _, hostnameOk = server.hosts[hostname]
	return hostOk || hostnameOk
}

func (server *Server) buildHosts() {
	if server.hosts == nil {
		server.hosts = map[string]struct{}{}
	}
	for _, host := range server.Hosts {
		server.hosts[host] = struct{}{}
	}
}
