module github.com/ninedraft/gemax

go 1.20

replace tailscale.com/net/memnet => ./vend/tailscale.com/net/memnet

require (
	golang.org/x/exp v0.0.0-20230522175609-2e198f4a06a1
	golang.org/x/net v0.11.0
)
