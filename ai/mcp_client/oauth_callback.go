package mcp_client

import (
	"context"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/lmorg/ttyphoon/app"
)

type OAuthCallbackServer struct {
	server *http.Server
	result chan OAuthCallbackResult
	err    chan error
}

type OAuthCallbackResult struct {
	Code  string
	State string
}

func StartOAuthCallbackServer(redirectURI string) (*OAuthCallbackServer, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return nil, fmt.Errorf("invalid redirect URI: %w", err)
	}
	if u.Scheme != "http" {
		return nil, fmt.Errorf("redirect URI must use http loopback for automatic callback capture")
	}
	host := u.Hostname()
	if host != "127.0.0.1" && host != "localhost" {
		return nil, fmt.Errorf("redirect URI host must be localhost or 127.0.0.1")
	}
	if u.Port() == "" {
		return nil, fmt.Errorf("redirect URI must include a port")
	}

	path := u.Path
	if path == "" {
		path = "/"
	}

	result := make(chan OAuthCallbackResult, 1)
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		if code == "" || state == "" {
			http.Error(w, "Missing code or state", http.StatusBadRequest)
			return
		}

		_, _ = io.WriteString(w, "<html><body><h1>Authentication complete</h1><p>You can close this window and return to "+html.EscapeString(app.Name())+".</p><script>window.close();</script></body></html>")
		select {
		case result <- OAuthCallbackResult{Code: code, State: state}:
		default:
		}
	})

	listener, err := net.Listen("tcp", u.Host)
	if err != nil {
		return nil, fmt.Errorf("cannot listen on OAuth callback URI %s: %w", redirectURI, err)
	}

	server := &http.Server{Handler: mux}
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			errCh <- serveErr
		}
	}()

	return &OAuthCallbackServer{server: server, result: result, err: errCh}, nil
}

func (s *OAuthCallbackServer) Wait(timeout time.Duration) (*OAuthCallbackResult, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case res := <-s.result:
		return &res, nil
	case err := <-s.err:
		return nil, fmt.Errorf("OAuth callback server error: %w", err)
	case <-timer.C:
		return nil, fmt.Errorf("timed out waiting for OAuth callback")
	}
}

func (s *OAuthCallbackServer) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.server.Shutdown(ctx)
}
