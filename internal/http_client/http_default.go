//go:build !linux

package http_client

import (
	"github.com/Mmx233/BitSrunLoginGo/internal/config"
	"github.com/Mmx233/BitSrunLoginGo/internal/config/keys"
	"net/http"
)

func newClientForInterface(netIface string) (*http.Client, error) {
	if netIface != "" {
		config.Logger.WithField(keys.LogComponent, "http_client").Warnf("Interface binding is not supported on this OS, ignoring interface %s", netIface)
	}

	transport := newDefaultTransport(config.Settings.Basic.SkipCertVerify)

	return &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}, nil
}