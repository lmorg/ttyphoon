package mcp_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
)

type FileTokenStore struct {
	path string
}

func NewFileTokenStore(path string) *FileTokenStore {
	return &FileTokenStore{path: path}
}

func (s *FileTokenStore) GetToken(ctx context.Context) (*client.Token, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, transport.ErrNoToken
		}
		return nil, fmt.Errorf("read token file: %w", err)
	}

	var tok client.Token
	if err := json.Unmarshal(b, &tok); err != nil {
		return nil, fmt.Errorf("parse token file: %w", err)
	}

	if tok.AccessToken == "" {
		return nil, transport.ErrNoToken
	}

	return &tok, nil
}

func (s *FileTokenStore) SaveToken(ctx context.Context, token *client.Token) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if token == nil {
		return fmt.Errorf("cannot save nil token")
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create token directory: %w", err)
	}

	b, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}

	if err := os.WriteFile(s.path, b, 0o600); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}

	return nil
}
