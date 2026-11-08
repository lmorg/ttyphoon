package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
)

type oauthWindowRuntime struct {
	types.DesktopRuntime
	external []string
	builtin  []string
	err      error
	closed   []string
}

func (r *oauthWindowRuntime) OpenURL(url string) error {
	r.external = append(r.external, url)
	return r.err
}

func (r *oauthWindowRuntime) OpenURLInNewWindow(url string) error {
	r.builtin = append(r.builtin, url)
	return r.err
}

func (r *oauthWindowRuntime) OpenManagedURLWindow(url string) (func(), error) {
	if err := r.OpenURLInNewWindow(url); err != nil {
		return nil, err
	}
	return func() { r.closed = append(r.closed, url) }, nil
}

func TestMcpOAuthWindowSelection(t *testing.T) {
	previous := config.Config.Ai.OAuthUseBuiltinWebview
	t.Cleanup(func() { config.Config.Ai.OAuthUseBuiltinWebview = previous })
	desktop := &oauthWindowRuntime{}
	ctx := runtime.WithRuntime(context.Background(), desktop)
	hooks := mcpOAuthWindowHooks(ctx)
	config.Config.Ai.OAuthUseBuiltinWebview = true
	for _, url := range []string{"https://first.example/authorize", "https://second.example/authorize"} {
		if err := hooks.OpenBrowser(url); err != nil {
			t.Fatal(err)
		}
	}
	if len(desktop.builtin) != 2 || len(desktop.external) != 0 {
		t.Fatal("built-in logins did not each request their own URL window")
	}
	config.Config.Ai.OAuthUseBuiltinWebview = false
	if err := hooks.OpenBrowser("https://external.example/authorize"); err != nil {
		t.Fatal(err)
	}
	if len(desktop.external) != 1 || len(desktop.builtin) != 2 {
		t.Fatal("external browser override was not honored")
	}
	if len(desktop.closed) != 0 {
		t.Fatal("authentication windows closed before success")
	}
	hooks.OnAuthenticated()
	hooks.OnAuthenticated()
	if len(desktop.closed) != 2 || desktop.closed[0] != desktop.builtin[0] || desktop.closed[1] != desktop.builtin[1] {
		t.Fatal("successful authentication did not close exactly its built-in windows")
	}
}

func TestMcpOAuthWindowErrors(t *testing.T) {
	previous := config.Config.Ai.OAuthUseBuiltinWebview
	t.Cleanup(func() { config.Config.Ai.OAuthUseBuiltinWebview = previous })
	for _, builtin := range []bool{false, true} {
		config.Config.Ai.OAuthUseBuiltinWebview = builtin
		if err := mcpOAuthWindowHooks(context.Background()).OpenBrowser("https://example.com"); !errors.Is(err, runtime.ErrUnavailable) {
			t.Fatalf("missing runtime error = %v", err)
		}

		failure := errors.New("authorization page could not be opened")
		ctx := runtime.WithRuntime(context.Background(), &oauthWindowRuntime{err: failure})
		if err := mcpOAuthWindowHooks(ctx).OpenBrowser("https://example.com"); !errors.Is(err, failure) {
			t.Fatalf("opener error = %v", err)
		}
	}
}

func TestMcpOAuthSuccessClosesOnlyOwningLoginWindow(t *testing.T) {
	previous := config.Config.Ai.OAuthUseBuiltinWebview
	t.Cleanup(func() { config.Config.Ai.OAuthUseBuiltinWebview = previous })
	config.Config.Ai.OAuthUseBuiltinWebview = true
	desktop := &oauthWindowRuntime{}
	ctx := runtime.WithRuntime(context.Background(), desktop)
	first, second := mcpOAuthWindowHooks(ctx), mcpOAuthWindowHooks(ctx)
	if err := first.OpenBrowser("https://first.example/login"); err != nil {
		t.Fatal(err)
	}
	if err := second.OpenBrowser("https://second.example/login"); err != nil {
		t.Fatal(err)
	}
	if err := desktop.OpenURLInNewWindow("https://notes.example"); err != nil {
		t.Fatal(err)
	}
	first.OnAuthenticated()
	if len(desktop.closed) != 1 || desktop.closed[0] != "https://first.example/login" {
		t.Fatal("authentication closed a window belonging to another flow")
	}
	second.OnAuthenticated()
	if len(desktop.closed) != 2 || desktop.closed[1] != "https://second.example/login" {
		t.Fatal("second authentication did not close its own login window")
	}
}
