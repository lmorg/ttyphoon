package lsp

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
)

// SessionKey identifies a language-server session by workspace root and language.
type SessionKey struct {
	WorkspaceRoot string
	LanguageID    string
}

// Manager owns one ServerProcess per workspace+language key.
// It is the single entry point for acquiring a transport to a language server.
type Manager struct {
	mu      sync.Mutex
	servers map[SessionKey]*ServerProcess
}

func NewManager() *Manager {
	return &Manager{servers: make(map[SessionKey]*ServerProcess)}
}

// GetOrStart returns the running ServerProcess for the given key.
// If none exists, it creates one from argv and starts it.
// argv must be non-empty; argv[0] is the executable.
func (m *Manager) GetOrStart(ctx context.Context, workspaceRoot, languageID string, argv []string) (*ServerProcess, error) {
	if m == nil {
		return nil, fmt.Errorf("lsp: manager is nil")
	}

	if err := validateArgv(argv); err != nil {
		return nil, err
	}

	key := SessionKey{WorkspaceRoot: workspaceRoot, LanguageID: languageID}

	m.mu.Lock()
	defer m.mu.Unlock()

	if sp, ok := m.servers[key]; ok {
		return sp, nil
	}

	sp, err := NewServerProcess(argv)
	if err != nil {
		return nil, err
	}

	if err := sp.Start(ctx); err != nil {
		return nil, err
	}

	m.servers[key] = sp
	return sp, nil
}

// Has reports whether a server is already running for the given key.
func (m *Manager) Has(workspaceRoot, languageID string) bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	_, ok := m.servers[SessionKey{WorkspaceRoot: workspaceRoot, LanguageID: languageID}]
	m.mu.Unlock()
	return ok
}

// ServersForWorkspace returns a snapshot of running servers for one workspace root.
func (m *Manager) ServersForWorkspace(workspaceRoot string) []*ServerProcess {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	servers := make([]*ServerProcess, 0, len(m.servers))
	for key, sp := range m.servers {
		if key.WorkspaceRoot == workspaceRoot {
			servers = append(servers, sp)
		}
	}

	return servers
}

// Stop shuts down the server for a given key and removes it.
func (m *Manager) Stop(workspaceRoot, languageID string) {
	if m == nil {
		return
	}

	key := SessionKey{WorkspaceRoot: workspaceRoot, LanguageID: languageID}

	m.mu.Lock()
	sp, ok := m.servers[key]
	if ok {
		delete(m.servers, key)
	}
	m.mu.Unlock()

	if ok {
		sp.Stop()
	}
}

// StopAll shuts down every managed server.
func (m *Manager) StopAll() {
	if m == nil {
		return
	}

	m.mu.Lock()
	servers := make(map[SessionKey]*ServerProcess, len(m.servers))
	for k, v := range m.servers {
		servers[k] = v
	}
	m.servers = make(map[SessionKey]*ServerProcess)
	m.mu.Unlock()

	for _, sp := range servers {
		sp.Stop()
	}
}

// ValidateConfig checks a Notes.LSP map for obviously bad entries.
// It returns a slice of error strings; nil means all entries look valid.
func ValidateConfig(lspConfig map[string][]string) []string {
	var errs []string
	for lang, argv := range lspConfig {
		if len(argv) == 0 {
			errs = append(errs, fmt.Sprintf("Notes.LSP[%q]: argv must not be empty", lang))
			continue
		}
		if strings.TrimSpace(argv[0]) == "" {
			errs = append(errs, fmt.Sprintf("Notes.LSP[%q]: argv[0] (executable) must not be blank", lang))
		}
	}
	return errs
}

func validateArgv(argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("lsp: argv must not be empty")
	}
	if strings.TrimSpace(argv[0]) == "" {
		return fmt.Errorf("lsp: argv[0] (executable) must not be blank")
	}
	return nil
}

// LookupArgv returns the argv from config for a given language id, or nil.
func LookupArgv(lspConfig map[string][]string, languageID string) []string {
	if argv, ok := lspConfig[languageID]; ok {
		return slices.Clone(argv)
	}
	return nil
}
