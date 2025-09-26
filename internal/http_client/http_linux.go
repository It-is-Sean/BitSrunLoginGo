//go:build linux

package http_client

import (
	"fmt"
	"github.com/Mmx233/BitSrunLoginGo/internal/config"
	"net"
	"net/http"
	"strings"
	"syscall"
)

func newClientForInterface(netIface string) (*http.Client, error) {
	transport := newDefaultTransport(config.Settings.Basic.SkipCertVerify)

	if netIface != "" {
		inter, err := net.InterfaceByName(netIface)
		if err != nil {
			return nil, fmt.Errorf("failed to get interface %s: %w", netIface, err)
		}

		addrs, err := inter.Addrs()
		if err != nil {
			return nil, fmt.Errorf("failed to get addresses for interface %s: %w", netIface, err)
		}

		var localAddr *net.TCPAddr
		for _, addr := range addrs {
			if strings.Contains(addr.String(), ".") {
				ip, err := net.ResolveTCPAddr("tcp", strings.Split(addr.String(), "/")[0]+":0")
				if err == nil {
					localAddr = ip
					break
				}
			}
		}

		if localAddr == nil {
			return nil, fmt.Errorf("no suitable IPv4 address found for interface %s", netIface)
		}

		dialer := &net.Dialer{
			Timeout:   config.Timeout,
			KeepAlive: 30 * config.Timeout, // KeepAlive is not set in original code, adding a reasonable default
			LocalAddr: localAddr,
			Control: func(network, address string, c syscall.RawConn) error {
				var opErr error
				fn := func(fd uintptr) {
					opErr = syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, netIface)
				}
				if err := c.Control(fn); err != nil {
					return err
				}
				return opErr
			},
		}
		transport.DialContext = dialer.DialContext
	}

	return &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}, nil
}