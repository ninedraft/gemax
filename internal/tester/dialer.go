package tester

import (
	"context"
	"crypto/tls"
	"io"
	"io/fs"
	"net"
	"path"
	"strings"
	"time"
)

// DialFS implements a fake dialer, which can stream fs files by stub connections.
type DialFS struct {
	Prefix string
	FS     fs.FS
}

// Dial returns a stub connection, which streams bytes from domain file.
func (dialer *DialFS) Dial(_ context.Context, host string, _ *tls.Config) (net.Conn, error) {
	host = strings.TrimPrefix(host, dialer.Prefix)
	var hostNoPort, _, errSplit = net.SplitHostPort(host)
	if errSplit != nil {
		return nil, errSplit
	}
	var file, errOpen = dialer.FS.Open(path.Join(dialer.Prefix, hostNoPort))
	if errOpen != nil {
		return nil, errOpen
	}
	return &staticConn{ReadCloser: file}, nil
}

type staticConn struct {
	io.ReadCloser
}

func (conn *staticConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (conn *staticConn) LocalAddr() net.Addr {
	panic("not implemented")
}

func (conn *staticConn) RemoteAddr() net.Addr {
	panic("not implemented")
}

func (conn *staticConn) SetDeadline(t time.Time) error {
	return nil
}

func (conn *staticConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (conn *staticConn) SetWriteDeadline(t time.Time) error {
	return nil
}
