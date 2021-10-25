package server

import (
	"crypto/tls"
	"net"
	"os"
	"strconv"

	"github.com/hacdias/eagle/config"
	"tailscale.com/client/tailscale"
	"tailscale.com/tsnet"
)

func (s *Server) getTailscaleListener(c *config.Tailscale) (net.Listener, error) {
	addr := ":" + strconv.Itoa(c.Port)

	_ = os.Setenv("TS_AUTHKEY", c.AuthKey)
	_ = os.Setenv("TAILSCALE_USE_WIP_CODE", "true")

	server := &tsnet.Server{
		Hostname: c.Hostname,
	}

	if c.Logging {
		server.Logf = s.Named("tailscale").Infof
	} else {
		server.Logf = func(format string, args ...interface{}) {}
	}

	ln, err := server.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	if addr == ":443" {
		ln = tls.NewListener(ln, &tls.Config{
			GetCertificate: tailscale.GetCertificate,
		})
	}

	return ln, nil
}
