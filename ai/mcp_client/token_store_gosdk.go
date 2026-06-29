package mcp_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
)

type filePersistingTokenSource struct {
	path string
	base oauth2.TokenSource
}

// NewFilePersistingTokenSource returns an oauth2.TokenSource that attempts to
// read an existing token from `path` when `base` is nil, otherwise delegates
// to `base` and persists tokens returned by it to `path`.
func NewFilePersistingTokenSource(path string, base oauth2.TokenSource) oauth2.TokenSource {
	return &filePersistingTokenSource{path: path, base: base}
}

func (s *filePersistingTokenSource) Token() (*oauth2.Token, error) {
	if s.base == nil {
		t, err := readTokenFile(s.path)
		if err != nil {
			return nil, err
		}
		return t, nil
	}

	t, err := s.base.Token()
	if err != nil {
		return nil, err
	}

	if scope, _ := t.Extra("scope").(string); scope != "" {
		log.Printf("MCP OAuth: token obtained granted_scope=%q token_type=%q expiry=%q", scope, t.TokenType, t.Expiry.Format(time.RFC3339))
	} else {
		log.Printf("MCP OAuth: token obtained granted_scope=<none in response> token_type=%q expiry=%q", t.TokenType, t.Expiry.Format(time.RFC3339))
	}

	// best-effort persist
	_ = writeTokenFile(s.path, t)
	return t, nil
}

func readTokenFile(path string) (*oauth2.Token, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("no token")
		}
		return nil, err
	}

	var tok oauth2.Token
	if err := json.Unmarshal(b, &tok); err != nil {
		return nil, fmt.Errorf("parse token file: %w", err)
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("no access token")
	}
	return &tok, nil
}

func writeTokenFile(path string, token *oauth2.Token) error {
	if token == nil {
		return fmt.Errorf("nil token")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create token directory: %w", err)
	}
	b, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}
	return nil
}
