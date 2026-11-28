// This file is added to the main package only by the offline CLI test overlay.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"os"
)

func init() {
	cert, err := os.ReadFile(os.Getenv("OFFLINE_ISSUE_CERT"))
	if err != nil {
		panic(err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(cert) {
		panic("invalid offline TLS certificate")
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{RootCAs: roots, ServerName: "example.com"}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != "api.github.com:443" {
			return nil, errors.New("unexpected offline dial address")
		}
		return (&net.Dialer{}).DialContext(ctx, network, os.Getenv("OFFLINE_ISSUE_ADDR"))
	}
	http.DefaultTransport = transport
}
