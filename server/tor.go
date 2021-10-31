package server

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/cretz/bine/tor"
)

func (s *Server) startTor(errCh chan error) error {
	key, err := getTorKey(s.Config.Tor.ConfDir)
	if err != nil {
		return err
	}

	s.log.Info("starting a Tor instance")
	cfg := &tor.StartConf{
		TempDataDirBase: os.TempDir(),
	}

	if s.Config.Tor.Logging {
		cfg.DebugWriter = os.Stdout
	}

	t, err := tor.Start(context.Background(), &tor.StartConf{
		TempDataDirBase: os.TempDir(),
	})
	if err != nil {
		return err
	}

	s.log.Info("creating and publishing the onion service ")

	// Wait at most a few minutes to publish the service
	listenCtx, listenCancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer listenCancel()

	// Create a v3 onion service to listen on any port but show as 80
	ln, err := t.Listen(listenCtx, &tor.ListenConf{
		Version3:    true,
		Key:         key,
		RemotePorts: []int{80},
	})
	if err != nil {
		t.Close()
		return err
	}

	s.onionAddress = "http://" + ln.String()

	srv := &http.Server{
		Handler:      s.makeRouter(true),
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
	}

	go func() {
		err = s.registerServer(srv, "tor")
		if err != nil {
			errCh <- err
			return
		}

		defer t.Close()
		s.log.Infof("tor listening on %s", ln.Addr().String())
		errCh <- srv.Serve(ln)

		// Clear onion address in case this error happens during runtime.
		s.onionAddress = ""
	}()

	return nil
}

func (s *Server) onionRedirHandler(w http.ResponseWriter, r *http.Request) {
	if s.onionAddress == "" {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	http.Redirect(w, r, s.onionAddress, http.StatusTemporaryRedirect)
}

func (s *Server) onionHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Onion-Location", s.onionAddress+r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func generateTorKey(keyPath string) (crypto.PrivateKey, error) {
	_, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}

	x509Encoded, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}

	pemEncoded := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509Encoded,
	})

	return key, os.WriteFile(keyPath, pemEncoded, 0644)
}

func readTorKey(keyPath string) (crypto.PrivateKey, error) {
	d, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(d)
	x509Encoded := block.Bytes

	return x509.ParsePKCS8PrivateKey(x509Encoded)
}

func getTorKey(path string) (crypto.PrivateKey, error) {
	keyPath := filepath.Join(path, "onion.pk")

	var (
		torKey crypto.PrivateKey
		err    error
	)

	if _, statErr := os.Stat(keyPath); statErr == nil {
		torKey, err = readTorKey(keyPath)
	} else if os.IsNotExist(statErr) {
		torKey, err = generateTorKey(keyPath)
	} else {
		return nil, statErr
	}

	return torKey, err
}
