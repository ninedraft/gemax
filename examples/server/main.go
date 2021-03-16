package main

import (
	"context"
	"crypto/tls"
	"embed"
	"fmt"
	"log"
	"time"

	gemax "github.com/ninedraft/gemax"
)

//go:embed index.gmi
var helloPage []byte

func main() {
	var addr = "localhost:1986"
	var ctx = context.Background()
	var server = &gemax.Server{
		Handler: func(ctx context.Context, rw gemax.ResponseWriter, req gemax.IncomingRequest) {
			log.Println("incoming request", req.URL())
			rw.Write(helloPage)
			fmt.Fprintf(rw, "\n%s\n", time.Now().Format(time.RubyDate))
		},
	}

	var listener, errListener = tls.Listen("tcp", addr, &tls.Config{
		Certificates: []tls.Certificate{
			loadCert(),
		},
	})
	if errListener != nil {
		panic(errListener)
	}
	log.Println("serving at", addr)
	var errServe = server.Serve(ctx, listener)
	if errServe != nil {
		panic(errServe)
	}
}

//go:embed certs/*
var certs embed.FS

func loadCert() tls.Certificate {
	var cert, errCertPEM = certs.ReadFile("certs/cert.pem")
	if errCertPEM != nil {
		panic(errCertPEM)
	}
	var key, errKeyPEM = certs.ReadFile("certs/key.pem")
	if errKeyPEM != nil {
		panic(errKeyPEM)
	}
	var c, errPars = tls.X509KeyPair(cert, key)
	if errPars != nil {
		panic(errPars)
	}
	return c
}
