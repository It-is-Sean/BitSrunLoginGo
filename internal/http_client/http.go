package http_client

import (
	"crypto/tls"
	"net/http"
)

// NewClient creates a new http.Client.
// If netIface is not empty, it will try to bind the client to the specified network interface.
func NewClient(netIface string) (*http.Client, error) {
	if netIface == "" {
		return newClientForInterface("")
	}
	return newClientForInterface(netIface)
}

func newDefaultTransport(skipCertVerify bool) *http.Transport {
	return &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipCertVerify},
		// You can add other default transport settings here
	}
}