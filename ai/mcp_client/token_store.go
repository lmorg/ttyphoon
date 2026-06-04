package mcp_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

type FileTokenStore struct {
	path string
}

func NewFileTokenStore(path string) *FileTokenStore {
	return &FileTokenStore{path: path}
}

func (s *FileTokenStore) GetToken(ctx context.Context) (*oauth2.Token, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("token not found")
		}
		return nil, fmt.Errorf("read token file: %w", err)
	}

	var tok oauth2.Token
	if err := json.Unmarshal(b, &tok); err != nil {
		return nil, fmt.Errorf("parse token file: %w", err)
	}

	return &tok, nil
}

func (s *FileTokenStore) StoreToken(ctx context.Context, token *oauth2.Token) error {
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
