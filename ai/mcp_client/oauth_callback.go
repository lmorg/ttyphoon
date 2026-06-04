package mcp_client

import (
	"context"
	"fmt"
	"html"
	"io"
	"log"
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
	log.Printf("MCP OAuth: starting callback server redirect_uri=%q", redirectURI)
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
		log.Printf("MCP OAuth: callback request received remote=%q path=%q host=%q", r.RemoteAddr, r.URL.Path, r.Host)
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		if code == "" || state == "" {
			log.Printf("MCP OAuth: callback missing code or state")
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
	log.Printf("MCP OAuth: callback server listening on %q", listener.Addr().String())

	server := &http.Server{Handler: mux}
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			log.Printf("MCP OAuth: callback server serve error: %v", serveErr)
			errCh <- serveErr
		}
	}()

	return &OAuthCallbackServer{server: server, result: result, err: errCh}, nil
}

func (s *OAuthCallbackServer) Wait(timeout time.Duration) (*OAuthCallbackResult, error) {
	log.Printf("MCP OAuth: waiting for callback timeout=%s", timeout)
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case res := <-s.result:
		log.Printf("MCP OAuth: callback wait completed state_present=%t code_len=%d", res.State != "", len(res.Code))
		return &res, nil
	case err := <-s.err:
		log.Printf("MCP OAuth: callback wait failed: %v", err)
		return nil, fmt.Errorf("OAuth callback server error: %w", err)
	case <-timer.C:
		log.Printf("MCP OAuth: callback wait timed out")
		return nil, fmt.Errorf("timed out waiting for OAuth callback")
	}
}

func (s *OAuthCallbackServer) Close() {
	log.Printf("MCP OAuth: shutting down callback server")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.server.Shutdown(ctx)
}
