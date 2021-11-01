package server

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/hacdias/eagle/v2/logging"
	"tailscale.com/client/tailscale"
	"tailscale.com/tsnet"
)

func (s *Server) getTailscaleListener() (net.Listener, error) {
	c := s.Config.Tailscale
	addr := ":" + strconv.Itoa(c.Port)

	_ = os.Setenv("TS_AUTHKEY", c.AuthKey)
	_ = os.Setenv("TAILSCALE_USE_WIP_CODE", "true")

	server := &tsnet.Server{
		Hostname: c.Hostname,
	}

	if c.Logging {
		server.Logf = logging.S().Named("tailscale").Infof
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

func (s *Server) startTailscaleServer(errCh chan error) error {
	ln, err := s.getTailscaleListener()
	if err != nil {
		return err
	}

	router := s.makeRouter(false)
	srv := &http.Server{Handler: router}

	s.registerServer(srv, "tailscale")

	go func() {
		s.log.Infof("tailscale listening on %s", ln.Addr().String())
		errCh <- srv.Serve(ln)
	}()

	return nil
}
