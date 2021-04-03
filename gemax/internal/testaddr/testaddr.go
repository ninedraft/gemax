// Package testaddr selects free port for testing purposes.
package testaddr

import (
	"fmt"
	"net"
)

const localAddr = "localhost:0"

// Addr returns free address in form of "localhost:$FREE_PORT",
// which can be used for test servers.
func Addr() string {
	var listener, errListener = net.Listen("tcp", localAddr)
	if errListener != nil {
		var msg = fmt.Sprintf("resolving address %q: %v", localAddr, errListener)
		panic(msg)
	}
	var addr = listener.Addr()
	_ = listener.Close()
	var _, port, errSplit = net.SplitHostPort(addr.String())
	if errSplit != nil {
		var msg = fmt.Sprintf("parsing address %q: %v", addr, errSplit)
		panic(msg)
	}
	return "localhost:" + port
}
