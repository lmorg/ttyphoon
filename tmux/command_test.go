package tmux

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	runebuf "github.com/lmorg/ttyphoon/utils/rune_buf"
	"github.com/lmorg/ttyphoon/utils/workspace"
)

func TestCommandLine(t *testing.T) {
	expected := `list-windows  -F 'Name:#{window_name}|||Id:#{window_id}|||Width:#{window_width}|||Height:#{window_height}|||Active:#{?window_active,true,false}'`

	var _test_CMD_LIST_WINDOWS = "list-windows"

	type _test_CMD_LIST_WINDOWS_T struct {
		Name       string `tmux:"window_name"`
		Id         string `tmux:"window_id"`
		Width      int    `tmux:"window_width"`
		Height     int    `tmux:"window_height"`
		Active     bool   `tmux:"?window_active,true,false"`
		panes      map[string]*PaneT
		activePane *PaneT
	}

	actual := string(mkCmdLine(
		_test_CMD_LIST_WINDOWS,
		reflect.TypeOf(_test_CMD_LIST_WINDOWS_T{}),
	))

	if expected != actual {
		t.Errorf("Incorrect string returned:")
		t.Logf("  Expected: %s", expected)
		t.Logf("  Actual:   %s", actual)
		t.Logf("  exp byte: %v", []byte(expected))
		t.Logf("  act byte: %v", []byte(actual))
	}
}

func TestPaneListReadsPerPaneZoomFlag(t *testing.T) {
	command := string(mkCmdLine(CMD_LIST_PANES, reflect.TypeOf(paneInfo{}), "-s"))
	if !strings.Contains(command, "Zoomed:#{?pane_zoomed_flag,true,false}") {
		t.Fatalf("list-panes command does not query the pane-specific zoom flag: %s", command)
	}
	if strings.Contains(command, "pane_zoomed}") || strings.Contains(command, "window_zoomed_flag") {
		t.Fatalf("list-panes command uses a non-pane-specific zoom flag: %s", command)
	}
}

func TestQuoteStringEscapesTmuxArgument(t *testing.T) {
	if got, want := workspace.QuoteString(`SSH_AUTH_SOCK=/tmp/agent\"value`), `"SSH_AUTH_SOCK=/tmp/agent\\\"value"`; got != want {
		t.Fatalf("quoteString() = %q, want %q", got, want)
	}
}

type paneNavigationTestRenderer struct {
	layoutChangeTestRenderer
	tmux        *Tmux
	maximizedID string
}

func (renderer *paneNavigationTestRenderer) ZoomPaneAndMaximize(paneID string) error {
	if err := renderer.tmux.ZoomPane(paneID); err != nil {
		return err
	}
	renderer.maximizedID = paneID
	renderer.RefreshWindowList()
	return nil
}

func TestPaneNavigationAndZoomBindingsRefreshRenderer(t *testing.T) {
	for _, test := range []struct{ note, command string }{
		{"Move to the previously active pane", "select-pane -l"},
		{"Zoom the active pane", "resize-pane -t %7 -Z"},
	} {
		t.Run(test.note, func(t *testing.T) {
			handler, ok := defaultFnKeys[test.note]
			if !ok {
				t.Fatalf("missing binding for %q", test.note)
			}
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = reader.Close()
				_ = writer.Close()
			})
			renderer := &paneNavigationTestRenderer{layoutChangeTestRenderer: layoutChangeTestRenderer{refreshed: make(chan struct{}, 1)}}
			tmux := &Tmux{tty: writer, resp: make(chan *tmuxResponseT, 1), wins: newWindowMap(), panes: newPaneMap(), renderer: renderer}
			renderer.tmux = tmux
			win := &WindowT{id: "@1", active: true, panes: newPaneMap()}
			pane := &PaneT{id: "%7", windowId: win.id, active: true}
			tmux.wins.Set(win.id, win)
			tmux.panes.Set(pane.id, pane)
			win.panes.Set(pane.id, pane)
			commands := make(chan string, 1)
			go func() {
				scanner := bufio.NewScanner(reader)
				if scanner.Scan() {
					commands <- scanner.Text()
					tmux.resp <- &tmuxResponseT{}
				}
			}()
			if err := handler(tmux); err != nil {
				t.Fatal(err)
			}
			if got := <-commands; got != test.command {
				t.Fatalf("binding command = %q, want %q", got, test.command)
			}
			if test.note == "Zoom the active pane" && renderer.maximizedID != pane.id {
				t.Fatalf("maximize request = %q, want %q", renderer.maximizedID, pane.id)
			}
			select {
			case <-renderer.refreshed:
			case <-time.After(time.Second):
				t.Fatal("binding did not refresh renderer")
			}
		})
	}
}

func TestSplitWindowUsesWorkspaceDirectoryWhenSupplied(t *testing.T) {
	previous := config.Config.Workspaces
	t.Cleanup(func() { config.Config.Workspaces = previous })
	t.Setenv("TTYPHOON_TEST_SPLIT_ROOT", "/tmp/workspace root")
	for _, test := range []struct {
		name     string
		settings *workspace.WorkspaceT
		suffix   string
	}{
		{name: "no workspace entry", settings: &workspace.WorkspaceT{
			Defaults:  workspace.TermT{Envs: map[string]string{}},
			Terminals: map[string]workspace.TermT{},
		}},
		{name: "workspace directory", settings: &workspace.WorkspaceT{
			Defaults: workspace.TermT{Envs: map[string]string{}},
			Terminals: map[string]workspace.TermT{
				"project": {Pwd: "/tmp/workspace", Envs: map[string]string{}},
			},
		}, suffix: ` -c "/tmp/workspace"`},
		{name: "named override", settings: &workspace.WorkspaceT{
			Defaults: workspace.TermT{Envs: map[string]string{}},
			Terminals: map[string]workspace.TermT{
				"project": {Pwd: `$TTYPHOON_TEST_SPLIT_ROOT/"quoted"`, Envs: map[string]string{}},
			},
		}, suffix: ` -c "/tmp/workspace root/\"quoted\""`},
	} {
		for _, flag := range []string{"-h", "-v"} {
			t.Run(test.name+flag, func(t *testing.T) {
				config.Config.Workspaces = test.settings
				reader, writer, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
				tmux := &Tmux{tty: writer, resp: make(chan *tmuxResponseT, 1), wins: newWindowMap()}
				win := &WindowT{id: "@1", name: "project", active: true}
				tmux.wins.Set(win.id, win)
				commands := make(chan string, 1)
				go func() {
					scanner := bufio.NewScanner(reader)
					if scanner.Scan() {
						commands <- scanner.Text()
						tmux.resp <- &tmuxResponseT{}
					}
				}()
				if err := _fnKeySplitWindow(tmux, flag); err != nil {
					t.Fatal(err)
				}
				if got, want := <-commands, "split-window "+flag+" -P -F '#{pane_id}'"+test.suffix; got != want {
					t.Fatalf("split command = %q, want %q", got, want)
				}
			})
		}
	}
}

func TestSplitWindowPassesAndSeedsActivePaneEnvironment(t *testing.T) {
	previous := config.Config.Workspaces
	config.Config.Workspaces = &workspace.WorkspaceT{
		Defaults:  workspace.TermT{Envs: map[string]string{}},
		Terminals: map[string]workspace.TermT{"project": {Envs: map[string]string{}}},
	}
	t.Cleanup(func() { config.Config.Workspaces = previous })
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
	tmux := &Tmux{tty: writer, resp: make(chan *tmuxResponseT, 1), wins: newWindowMap(), panes: newPaneMap()}
	win := &WindowT{id: "@1", name: "project", active: true, panes: newPaneMap()}
	tmux.wins.Set(win.id, win)
	tmux.activeWindow = win
	source := &PaneT{id: "%7", windowId: win.id, tmux: tmux, active: true}
	source.term = &shellCloneTestTerm{paneGeometryTestTerm: paneGeometryTestTerm{pane: source}}
	child := &PaneT{id: "%8", windowId: win.id, tmux: tmux}
	childTerm := &shellCloneTestTerm{paneGeometryTestTerm: paneGeometryTestTerm{pane: child}}
	child.term = childTerm
	win.activePane = source
	win.panes.Set(source.id, source)
	tmux.panes.Set(source.id, source)
	tmux.panes.Set(child.id, child)
	commands := make(chan string, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		if scanner := bufio.NewScanner(reader); scanner.Scan() {
			commands <- scanner.Text()
			tmux.resp <- &tmuxResponseT{Message: [][]byte{[]byte(child.id)}}
		}
	}()
	if err := _fnKeySplitWindow(tmux, "-h"); err != nil {
		t.Fatal(err)
	}
	command := <-commands
	if !strings.Contains(command, `-e "CLONE_TEST=inherited value"`) {
		t.Fatalf("split command omitted pane environment: %q", command)
	}
	if !reflect.DeepEqual(childTerm.initialEnvVars, map[string]string{"CLONE_TEST": "inherited value"}) {
		t.Fatalf("split pane initial environment = %#v", childTerm.initialEnvVars)
	}
	_ = writer.Close()
	<-done
}

func TestWorkspaceStartSeedsApplicationAndWorkspaceEnvironment(t *testing.T) {
	previous := config.Config.Workspaces
	config.Config.Workspaces = &workspace.WorkspaceT{
		Defaults: workspace.TermT{Envs: map[string]string{"DEFAULT_VALUE": "default"}},
		Terminals: map[string]workspace.TermT{
			"project": {Pwd: "/workspace/project", Envs: map[string]string{"PROJECT_VALUE": "workspace"}},
		},
	}
	t.Cleanup(func() { config.Config.Workspaces = previous })
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
	tmux := &Tmux{
		tty: writer, resp: make(chan *tmuxResponseT, 1), wins: newWindowMap(), panes: newPaneMap(),
		renderer: &paneCloseTestRenderer{}, initialEnvVars: map[string]string{"APP_VALUE": "app"},
	}
	child := &PaneT{id: "%8", tmux: tmux}
	childTerm := &shellCloneTestTerm{paneGeometryTestTerm: paneGeometryTestTerm{pane: child}}
	child.term = childTerm
	tmux.panes.Set(child.id, child)
	commands := make(chan string, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		if scanner := bufio.NewScanner(reader); scanner.Scan() {
			commands <- scanner.Text()
			tmux.resp <- &tmuxResponseT{Message: [][]byte{[]byte(child.id)}}
		}
	}()

	tmux.WorkspaceStart("project")
	command := <-commands
	for _, expected := range []string{
		`-e "APP_VALUE=app"`,
		`-e "DEFAULT_VALUE=default"`,
		`-e "PROJECT_VALUE=workspace"`,
		`-c "/workspace/project"`,
	} {
		if !strings.Contains(command, expected) {
			t.Errorf("workspace command %q does not contain %q", command, expected)
		}
	}
	wantEnvironment := map[string]string{
		"APP_VALUE":     "app",
		"DEFAULT_VALUE": "default",
		"PROJECT_VALUE": "workspace",
	}
	if !reflect.DeepEqual(childTerm.initialEnvVars, wantEnvironment) {
		t.Fatalf("workspace pane initial environment = %#v, want %#v", childTerm.initialEnvVars, wantEnvironment)
	}
	_ = writer.Close()
	<-done
}

func TestNewWindowInheritsAndSeedsActivePaneEnvironment(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
	tmux := &Tmux{tty: writer, resp: make(chan *tmuxResponseT, 1), wins: newWindowMap(), panes: newPaneMap()}
	win := &WindowT{id: "@1", active: true, panes: newPaneMap()}
	tmux.wins.Set(win.id, win)
	tmux.activeWindow = win
	source := &PaneT{id: "%7", windowId: win.id, tmux: tmux, active: true}
	source.term = &shellCloneTestTerm{paneGeometryTestTerm: paneGeometryTestTerm{pane: source}}
	child := &PaneT{id: "%8", windowId: win.id, tmux: tmux}
	childTerm := &shellCloneTestTerm{paneGeometryTestTerm: paneGeometryTestTerm{pane: child}}
	child.term = childTerm
	win.activePane = source
	win.panes.Set(source.id, source)
	tmux.panes.Set(source.id, source)
	tmux.panes.Set(child.id, child)
	commands := make(chan string, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		if scanner := bufio.NewScanner(reader); scanner.Scan() {
			commands <- scanner.Text()
			tmux.resp <- &tmuxResponseT{Message: [][]byte{[]byte(child.id)}}
		}
	}()
	if err := tmux.newWindowWithInitialEnvironment(); err != nil {
		t.Fatal(err)
	}
	if command := <-commands; !strings.Contains(command, `-e "CLONE_TEST=inherited value"`) {
		t.Fatalf("new-window command omitted pane environment: %q", command)
	}
	if !reflect.DeepEqual(childTerm.initialEnvVars, map[string]string{"CLONE_TEST": "inherited value"}) {
		t.Fatalf("new window initial environment = %#v", childTerm.initialEnvVars)
	}
	_ = writer.Close()
	<-done
}

func TestZoomPaneTargetsRequestedPane(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
	})
	tmux := &Tmux{tty: writer, resp: make(chan *tmuxResponseT, 1), panes: newPaneMap()}
	tmux.panes.Set("%17", &PaneT{id: "%17"})
	command := make(chan string, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for scanner := bufio.NewScanner(reader); scanner.Scan(); {
			command <- scanner.Text()
			tmux.resp <- &tmuxResponseT{}
		}
	}()
	if err := tmux.ZoomPane("%17"); err != nil {
		t.Fatalf("zoom pane: %v", err)
	}
	if got := <-command; got != "resize-pane -t %17 -Z" {
		t.Fatalf("zoom command = %q; want requested pane ID", got)
	}
	_ = writer.Close()
	<-done
}

func TestPaneZoomStateTracksTmuxMetadata(t *testing.T) {
	tmux := &Tmux{wins: newWindowMap(), panes: newPaneMap()}
	win := &WindowT{id: "@1", panes: newPaneMap()}
	tmux.wins.Set(win.id, win)
	pane := &PaneT{id: "%17", windowId: win.id, tmux: tmux}
	tmux.panes.Set(pane.id, pane)
	win.panes.Set(pane.id, pane)

	info := &paneInfo{Id: pane.id, WindowId: win.id, Zoomed: true}
	info.updatePane(tmux)
	if !pane.IsZoomed() {
		t.Fatal("pane metadata did not retain tmux zoom state")
	}
}

type layoutChangeTestRenderer struct {
	types.Renderer
	refreshed chan struct{}
	release   <-chan struct{}
}

type paneCloseTestRenderer struct{ types.Renderer }

func (*paneCloseTestRenderer) DisplayNotification(types.NotificationType, string) {}
func (*paneCloseTestRenderer) RefreshWindowList()                                 {}

type paneCloseRequestTestRenderer struct {
	types.Renderer
	request func(string, string) error
}

func (renderer *paneCloseRequestTestRenderer) ConfirmKillPane(paneID, paneName string) error {
	return renderer.request(paneID, paneName)
}

func TestKillPaneHotkeyUsesButtonClosePolicy(t *testing.T) {
	for _, finished := range []bool{false, true} {
		t.Run(fmt.Sprintf("finished=%t", finished), func(t *testing.T) {
			tmux := &Tmux{wins: newWindowMap(), panes: newPaneMap()}
			tmux.readerDead.Store(true)
			win := &WindowT{id: "@1", active: true, panes: newPaneMap()}
			pane := &PaneT{id: "%7", title: "Selected shell", windowId: win.id, active: true}
			pane.dead.Store(finished)
			tmux.wins.Set(win.id, win)
			tmux.panes.Set(pane.id, pane)
			win.panes.Set(pane.id, pane)
			calls := 0
			requestErr := fmt.Errorf("synthetic close request error")
			tmux.renderer = &paneCloseRequestTestRenderer{request: func(paneID, paneName string) error {
				calls++
				if paneID != pane.id || paneName != pane.title {
					t.Fatalf("close request = %q, %q; want selected pane ID and name", paneID, paneName)
				}
				return requestErr
			}}
			if err := fnKeyKillPane(tmux); err != requestErr || calls != 1 {
				t.Fatalf("hotkey error = %v, calls = %d; want delegated error and one request", err, calls)
			}
			if tmux.panes.Get(pane.id) != pane || pane.closed {
				t.Fatal("hotkey removed the pane before the shared close policy handled it")
			}
			tmux.renderer = &paneCloseTestRenderer{}
			if err := fnKeyKillPane(tmux); err == nil || err.Error() != "pane close confirmation is unavailable" {
				t.Fatalf("missing confirmation provider error = %v; want safe failure", err)
			}
		})
	}
}

func TestKillPanePrunesOnlyPanesMissingFromTmuxSnapshot(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
	})

	tmux := &Tmux{
		tty:      writer,
		resp:     make(chan *tmuxResponseT, 1),
		wins:     newWindowMap(),
		panes:    newPaneMap(),
		renderer: &paneCloseTestRenderer{},
	}
	win := &WindowT{id: "@1", panes: newPaneMap()}
	tmux.wins.Set(win.id, win)
	tmux.activeWindow = win

	removedPane := &PaneT{id: "%1", windowId: win.id, active: true, tmux: tmux, buf: runebuf.New()}
	survivingPane := &PaneT{id: "%2", windowId: win.id, active: false, tmux: tmux, buf: runebuf.New()}
	tmux.panes.Set(removedPane.id, removedPane)
	tmux.panes.Set(survivingPane.id, survivingPane)
	win.panes.Set(removedPane.id, removedPane)
	win.panes.Set(survivingPane.id, survivingPane)
	win.activePane = removedPane

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		for scanner := bufio.NewScanner(reader); scanner.Scan(); {
			line := scanner.Text()
			switch {
			case strings.HasPrefix(line, "kill-pane "):
				tmux.resp <- &tmuxResponseT{}
			case strings.HasPrefix(line, "list-panes "):
				tmux.resp <- &tmuxResponseT{Message: [][]byte{[]byte("Id:%2|||Width:80|||Height:24|||PosLeft:40|||PosTop:0|||PosRight:80|||PosBottom:24|||Active:true|||Dead:false|||WindowId:@1|||WinActive:true|||AtBottom:false|||Path:/tmp|||PathStart:/tmp|||Zoomed:false")}}
			default:
				tmux.resp <- &tmuxResponseT{IsErr: true, Message: [][]byte{[]byte("unexpected test command: " + line)}}
			}
		}
	}()

	if finished, err := tmux.PaneCommandFinished(removedPane.id); err != nil || finished {
		t.Fatalf("running pane state = %t, %v; want unfinished", finished, err)
	}
	if err := tmux.KillPane(removedPane.id); err != nil {
		t.Fatalf("confirmed pane kill failed: %v", err)
	}
	if got := tmux.panes.Get(removedPane.id); got != nil {
		t.Fatalf("removed pane remains in tmux pane map: %v", got.id)
	}
	if got := win.panes.Get(removedPane.id); got != nil {
		t.Fatalf("removed pane remains in window pane map: %v", got.id)
	}
	if !removedPane.closed {
		t.Fatal("pane missing from tmux's snapshot was not closed")
	}
	if survivingPane.closed || tmux.panes.Get(survivingPane.id) != survivingPane || win.panes.Get(survivingPane.id) != survivingPane {
		t.Fatal("pane still present in tmux's snapshot was incorrectly closed")
	}
	_ = writer.Close()
	<-serverDone
}

func TestPaneMapEachUsesStableSnapshot(t *testing.T) {
	panes := newPaneMap()
	first := &PaneT{id: "%1"}
	second := &PaneT{id: "%2"}
	inserted := &PaneT{id: "%3"}
	panes.Set(first.id, first)
	panes.Set(second.id, second)

	seen := make(map[string]bool)
	for pane := range panes.Each() {
		seen[pane.id] = true
		panes.Delete(second.id)
		panes.Set(inserted.id, inserted)
	}
	if len(seen) != 2 || !seen[first.id] || !seen[second.id] || seen[inserted.id] {
		t.Fatalf("iteration did not retain its starting snapshot: %v", seen)
	}
}

func (renderer *layoutChangeTestRenderer) RefreshWindowList() {
	renderer.refreshed <- struct{}{}
	if renderer.release != nil {
		<-renderer.release
	}
}

func TestPaneResizeDoesNotEchoKnownTmuxGeometry(t *testing.T) {
	tmux := &Tmux{}
	tmux.readerDead.Store(true)
	pane := &PaneT{id: "%7", width: 80, height: 24, tmux: tmux}
	if err := pane.Resize(&types.XY{X: 80, Y: 24}); err != nil {
		t.Fatalf("unchanged geometry attempted a tmux command: %v", err)
	}
	if err := pane.Resize(&types.XY{X: 100, Y: 24}); err == nil {
		t.Fatal("a requested size change did not attempt a tmux command")
	}
}

type paneGeometryTestTerm struct {
	types.Term
	pane      *PaneT
	resizeErr error
	resizes   int
	onFocus   func()
}

func (term *paneGeometryTestTerm) MakeVisible(bool) {}
func (term *paneGeometryTestTerm) SetFocus(bool) {
	if term.onFocus != nil {
		term.onFocus()
	}
}
func (term *paneGeometryTestTerm) Resize(size *types.XY) {
	term.resizes++
	term.resizeErr = term.pane.Resize(size)
}

func TestPaneInfoInitialWorkingDirectory(t *testing.T) {
	for _, test := range []struct {
		name, path, start, want string
		dead                    bool
	}{
		{"already finished", "", "/work/project", "/work/project", true},
		{"start path takes precedence", "/temporary/process/path", "/work/project", "/work/project", false},
		{"missing start path", "/work/project", "", "/work/project", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			info := &paneInfo{Path: test.path, PathStart: test.start, Dead: test.dead}
			if got := info.workingDirectory("", true); got != test.want {
				t.Fatalf("initial directory = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPaneInfoPreservesDirectoryWhenPaneFinishes(t *testing.T) {
	for _, test := range []struct {
		name, previous, path, start, want string
		dead                              bool
	}{
		{"finished without current path", "/work/project", "", "/work/project", "/work/project", true},
		{"finished with unreliable current path", "/work/project", "/other/root", "/work/project", "/work/project", true},
		{"finished preserves reported directory", "/work/project/subdir", "", "/work/project", "/work/project/subdir", true},
		{"live missing current path", "/work/project", "", "/work/project", "/work/project", false},
		{"unknown directory falls back to start", "", "", "/work/project", "/work/project", true},
		{"live directory changes still update", "/work/project", "/work/another", "/work/project", "/work/another", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &Tmux{panes: newPaneMap(), wins: newWindowMap()}
			win := &WindowT{id: "@1", panes: newPaneMap()}
			client.wins.Set(win.id, win)
			pane := &PaneT{id: "%7", windowId: win.id, pwd: test.previous, tmux: client}
			pane.term = &paneGeometryTestTerm{pane: pane, onFocus: func() {
				if got := pane.Pwd(); got != test.want {
					t.Errorf("focus callback directory = %q, want %q", got, test.want)
				}
			}}
			client.panes.Set(pane.id, pane)
			win.panes.Set(pane.id, pane)
			info := &paneInfo{Id: pane.id, WindowId: win.id, Active: true, Dead: test.dead, Path: test.path, PathStart: test.start}
			info.updatePane(client)
			if got := pane.Pwd(); got != test.want {
				t.Fatalf("pane directory = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPaneInfoResizeUpdatesLocalTerminalWithoutEchoingTmux(t *testing.T) {
	tmux := &Tmux{panes: newPaneMap(), wins: newWindowMap()}
	tmux.readerDead.Store(true)
	win := &WindowT{id: "@1", panes: newPaneMap()}
	tmux.wins.Set(win.id, win)
	pane := &PaneT{id: "%7", width: 40, height: 12, tmux: tmux}
	term := &paneGeometryTestTerm{pane: pane}
	pane.term = term
	tmux.panes.Set(pane.id, pane)
	info := &paneInfo{Id: pane.id, WindowId: win.id, Width: 80, Height: 24}
	info.updatePane(tmux)
	info.updatePane(tmux)
	if term.resizes != 2 || term.resizeErr != nil || pane.width != 80 || pane.height != 24 {
		t.Fatalf("geometry refresh echoed a resize: calls=%d error=%v size=%dx%d", term.resizes, term.resizeErr, pane.width, pane.height)
	}
}

func TestPaneInfoIgnoresMissingOrClosedOwnerWindow(t *testing.T) {
	for _, owner := range []string{"missing", "closed", "removed"} {
		for _, existing := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/existing=%t", owner, existing), func(t *testing.T) {
				tmux := &Tmux{panes: newPaneMap(), wins: newWindowMap()}
				win := &WindowT{id: "@39", panes: newPaneMap(), closed: owner == "closed"}
				if owner != "missing" {
					tmux.wins.Set(win.id, win)
				}
				pane := &PaneT{id: "%134", windowId: win.id, width: 80, height: 24, tmux: tmux}
				term := &paneGeometryTestTerm{pane: pane}
				pane.term = term
				if existing {
					tmux.panes.Set(pane.id, pane)
				}
				info := &paneInfo{Id: pane.id, WindowId: win.id, Width: 100, Height: 40, Active: true}
				if owner == "removed" {
					tmux.wins.Delete(win.id)
				}
				if got := info.updatePane(tmux); got != nil {
					t.Fatalf("stale pane update returned %+v", got)
				}
				if term.resizes != 0 || pane.width != 80 || pane.height != 24 || win.activePane != nil {
					t.Fatal("stale pane update mutated terminal or window state")
				}
				if !existing && tmux.panes.Get(pane.id) != nil {
					t.Fatal("stale pane update recreated a terminal after its window closed")
				}
			})
		}
	}
}

func TestActivePaneReconcilesClosedRemovedAndUnsetSelection(t *testing.T) {
	for _, state := range []string{"closed", "removed", "unset", "reported active"} {
		t.Run(state, func(t *testing.T) {
			win := &WindowT{id: "@1", panes: newPaneMap()}
			old := &PaneT{id: "%136", closed: state == "closed"}
			first := &PaneT{id: "%1", left: 0}
			selected := &PaneT{id: "%2", left: 40, active: true}
			win.panes.Set(first.id, first)
			win.panes.Set(selected.id, selected)
			if state != "unset" {
				win.activePane = old
			}
			if state == "closed" || state == "reported active" {
				win.panes.Set(old.id, old)
			}
			if got := win.ActivePane(); got != selected || win.activePane != selected {
				t.Fatalf("active pane = %v, want tmux-reported survivor", got)
			}
		})
	}
}

func TestActivePaneFallbackIsLocalAndEmptyWindowIsSafe(t *testing.T) {
	var uninitialized *Tmux
	if pane, err := uninitialized.ActivePane(); pane != nil || err == nil || err.Error() != "no open windows" {
		t.Fatalf("nil tmux active pane = %v, %v; want no-open-windows error", pane, err)
	}

	win := &WindowT{id: "@1", panes: newPaneMap(), activePane: &PaneT{id: "%closed", closed: true}}
	first := &PaneT{id: "%1", top: 0, left: 0}
	later := &PaneT{id: "%2", top: 10}
	win.panes.Set(later.id, later)
	win.panes.Set(first.id, first)
	if got := win.ActivePane(); got != first {
		t.Fatalf("local fallback = %v, want top-left surviving pane", got)
	}
	win.activePane = later
	if got := win.ActivePane(); got != later {
		t.Fatal("valid cached selection was not preserved")
	}
	win.panes.Delete(first.id)
	win.panes.Delete(later.id)
	if got := win.ActivePane(); got != nil || win.activePane != nil {
		t.Fatal("empty window retained a stale active pane")
	}
	win.closed = true
	win.panes.Set(first.id, first)
	if got := win.ActivePane(); got != nil {
		t.Fatal("closed window returned a pane")
	}
	tmux := &Tmux{activeWindow: win}
	if pane, err := tmux.ActivePane(); pane != nil || err == nil {
		t.Fatalf("unavailable active pane = %v, %v", pane, err)
	}
}

func TestPaneInfoPublishesSelectionBeforeFocusCallbacks(t *testing.T) {
	tmux := &Tmux{wins: newWindowMap(), panes: newPaneMap()}
	tmux.readerDead.Store(true)
	win := &WindowT{id: "@1", panes: newPaneMap(), activePane: &PaneT{id: "%136", closed: true}}
	tmux.wins.Set(win.id, win)
	pane := &PaneT{id: "%2", tmux: tmux}
	tmux.panes.Set(pane.id, pane)
	called := false
	pane.term = &paneGeometryTestTerm{pane: pane, onFocus: func() {
		called = true
		if win.panes.Get(pane.id) != pane || win.activePane != pane {
			t.Fatal("focus callback observed stale active-pane metadata")
		}
	}}
	info := &paneInfo{Id: pane.id, WindowId: win.id, Width: 80, Height: 24, Active: true}
	info.updatePane(tmux)
	if !called {
		t.Fatal("focus callback did not run")
	}
}

func TestActiveWindowSkipsClosedAndRemovedSelection(t *testing.T) {
	tmux := &Tmux{wins: newWindowMap()}
	old := &WindowT{id: "@39", closed: true}
	survivor := &WindowT{id: "@1", active: true}
	tmux.wins.Set(old.id, old)
	tmux.wins.Set(survivor.id, survivor)
	tmux.activeWindow = old
	if got, err := tmux.ActiveWindow(); err != nil || got != survivor {
		t.Fatalf("active window = %v, %v; want surviving window", got, err)
	}
	tmux.wins.Delete(survivor.id)
	if got, err := tmux.ActiveWindow(); got != nil || err == nil || tmux.activeWindow != nil {
		t.Fatalf("removed window still active: %v, %v", got, err)
	}
}

func TestEmptyPaneLayoutHasNilActiveTile(t *testing.T) {
	tmux := &Tmux{wins: newWindowMap(), panes: newPaneMap()}
	tmux.readerDead.Store(true)
	win := &WindowT{id: "@1", active: true, panes: newPaneMap()}
	tmux.wins.Set(win.id, win)
	tmux.activeWindow = win
	if layout := tmux.GetTermTiles(); layout.Active != nil || len(layout.Tiles) != 0 {
		t.Fatalf("empty window layout retained a typed-nil or stale tile: %+v", layout)
	}
	win.closed = true
	if layout := tmux.GetTermTiles(); layout.Active != nil || len(layout.Tiles) != 0 || len(layout.Tabs) != 0 {
		t.Fatalf("closed window layout retained tiles or tabs: %+v", layout)
	}
}

func TestLayoutChangeRefreshesPaneGeometry(t *testing.T) {
	renderer := &layoutChangeTestRenderer{refreshed: make(chan struct{}, 1)}
	tmuxCommandMap[_RESP_LAYOUT_CHANGE](&Tmux{renderer: renderer}, []byte("%layout-change @1 layout visible-layout *"))
	select {
	case <-renderer.refreshed:
	case <-time.After(time.Second):
		t.Fatal("tmux layout change did not refresh pane geometry")
	}
}

func TestLayoutChangeBurstCoalescesIntoOneTrailingRefresh(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	renderer := &layoutChangeTestRenderer{refreshed: make(chan struct{}, 128), release: release}
	tmux := &Tmux{renderer: renderer}
	_respLayoutChange(tmux, nil)
	select {
	case <-renderer.refreshed:
	case <-time.After(time.Second):
		t.Fatal("initial layout refresh did not start")
	}
	for range 100 {
		_respLayoutChange(tmux, nil)
	}
	tmux.layoutRefreshMu.Lock()
	running, pending := tmux.layoutRefreshRunning, tmux.layoutRefreshPending
	tmux.layoutRefreshMu.Unlock()
	if !running || !pending || len(renderer.refreshed) != 0 {
		t.Fatal("layout events were not coalesced behind the active refresh")
	}
	release <- struct{}{}
	select {
	case <-renderer.refreshed:
	case <-time.After(time.Second):
		t.Fatal("final layout refresh was lost")
	}
	tmux.layoutRefreshMu.Lock()
	pending = tmux.layoutRefreshPending
	tmux.layoutRefreshMu.Unlock()
	if pending || len(renderer.refreshed) != 0 {
		t.Fatal("layout burst scheduled more than one trailing refresh")
	}
}

func TestCompletedPaneStateAndCloseGuard(t *testing.T) {
	tmux := &Tmux{panes: newPaneMap()}
	pane := &PaneT{id: "%7", tmux: tmux}
	tmux.panes.Set(pane.id, pane)
	if pane.CommandFinished() {
		t.Fatal("running pane is marked completed")
	}
	if err := tmux.CloseCompletedPane(pane.id); err == nil || err.Error() != "pane command is still running" {
		t.Fatalf("running pane close error = %v", err)
	}
	if err := tmux.CloseCompletedPane("%missing"); err == nil {
		t.Fatal("missing pane was accepted")
	}
	pane.dead.Store(true)
	if !pane.CommandFinished() {
		t.Fatal("retained dead pane is not marked completed")
	}
	pane.dead.Store(false)
	for _, result := range []int32{1, 2} {
		pane.actionResult.Store(result)
		if !pane.CommandFinished() {
			t.Fatalf("interactive result %d is not marked completed", result)
		}
	}
}

func TestRespawnedPaneClearsCompletionState(t *testing.T) {
	tmux := &Tmux{panes: newPaneMap(), wins: newWindowMap()}
	win := &WindowT{id: "@1", panes: newPaneMap()}
	tmux.wins.Set(win.id, win)
	pane := &PaneT{id: "%7", tmux: tmux}
	pane.actionResult.Store(2)
	pane.autoCloseAt.Store(time.Now().Add(30 * time.Second).UnixMilli())
	tmux.panes.Set(pane.id, pane)
	info := &paneInfo{Id: pane.id, WindowId: win.id, Dead: true}
	info.updatePane(tmux)
	info.Dead = false
	info.updatePane(tmux)
	if pane.CommandFinished() || pane.AutoCloseAt() != 0 {
		t.Fatal("respawned pane still has completed-command state")
	}
}

type shellCloneTestTerm struct {
	paneGeometryTestTerm
	initialEnvVars map[string]string
}

func (*shellCloneTestTerm) GetEnvVars() map[string]string {
	return map[string]string{"CLONE_TEST": "inherited value"}
}

func (term *shellCloneTestTerm) SetInitialEnvVars(environment map[string]string) {
	term.initialEnvVars = cloneEnvironment(environment)
}

func TestClonePaneInheritsActivePaneDirectoryAndEnvironment(t *testing.T) {
	for _, test := range []struct {
		name, sourcePwd, command, want string
	}{
		{"live directory fallback", "", "", `split-window -v -P -F '#{pane_id}' -t %7 -c "#{pane_current_path}" -e "CLONE_TEST=inherited value"`},
		{"inherited directory", "/tmp/source dir", "", `split-window -v -P -F '#{pane_id}' -t %7 -c "#{pane_current_path}" -e "CLONE_TEST=inherited value"`},
		{"inherited command", "/tmp/source dir", "pwd", `split-window -v -k -P -F '#{pane_id}' -t %7 -c "#{pane_current_path}" -e "CLONE_TEST=inherited value"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
			client := &Tmux{tty: writer, resp: make(chan *tmuxResponseT, 1), wins: newWindowMap(), panes: newPaneMap(), renderer: &paneCloseTestRenderer{}}
			win := &WindowT{id: "@1", panes: newPaneMap()}
			client.wins.Set(win.id, win)
			source := &PaneT{id: "%7", windowId: win.id, tmux: client, pwd: test.sourcePwd, active: true}
			source.term = &shellCloneTestTerm{paneGeometryTestTerm: paneGeometryTestTerm{pane: source}}
			win.activePane = source
			win.panes.Set(source.id, source)
			cloned := &PaneT{id: "%8", windowId: win.id, tmux: client}
			clonedTerm := &shellCloneTestTerm{paneGeometryTestTerm: paneGeometryTestTerm{pane: cloned}}
			cloned.term = clonedTerm
			client.panes.Set(source.id, source)
			client.panes.Set(cloned.id, cloned)
			commands := make(chan string, 2)
			done := make(chan struct{})
			go func() {
				defer close(done)
				for scanner := bufio.NewScanner(reader); scanner.Scan(); {
					command := scanner.Text()
					commands <- command
					switch {
					case strings.HasPrefix(command, "split-window "):
						client.resp <- &tmuxResponseT{Message: [][]byte{[]byte("%8")}}
					case strings.HasPrefix(command, "list-panes "):
						client.resp <- &tmuxResponseT{Message: [][]byte{[]byte("Id:%8|||WindowId:@1|||Width:80|||Height:12|||Active:true")}}
					default:
						client.resp <- &tmuxResponseT{IsErr: true}
					}
				}
			}()
			var callback interactiveActionCallback
			if test.command != "" {
				callback = func([]byte, int, error) { t.Error("unexpected completion") }
			}
			if err := client.ClonePane(test.command, callback); err != nil {
				t.Fatal(err)
			}
			want := test.want
			if test.command != "" {
				if len(client.interactiveActions) != 1 {
					t.Fatal("command clone did not register completion")
				}
				for id, entry := range client.interactiveActions {
					t.Cleanup(func() { client.discardInteractiveAction(id) })
					if entry.paneID != cloned.id {
						t.Fatalf("completion pane = %q, want %q", entry.paneID, cloned.id)
					}
					want += " bash " + workspace.QuoteString(entry.scriptPath)
					script, err := os.ReadFile(entry.scriptPath)
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(string(script), quoteShellArgument((&url.URL{Path: test.sourcePwd}).EscapedPath())) {
						t.Fatalf("wrapper does not report source directory: %s", script)
					}
				}
			} else if len(client.interactiveActions) != 0 {
				t.Fatal("shell clone unexpectedly registered completion")
			}
			if got := <-commands; got != want {
				t.Fatalf("clone command = %q, want %q", got, want)
			}
			if got := <-commands; !strings.HasPrefix(got, "list-panes ") {
				t.Fatalf("clone did not refresh new pane metadata: %s", got)
			}
			if source.pwd != test.sourcePwd || cloned.CommandFinished() || cloned.AutoCloseAt() != 0 {
				t.Fatal("clone changed source directory or initialized completion state prematurely")
			}
			if !reflect.DeepEqual(clonedTerm.initialEnvVars, map[string]string{"CLONE_TEST": "inherited value"}) {
				t.Fatalf("clone initial environment = %#v", clonedTerm.initialEnvVars)
			}
			_ = writer.Close()
			<-done
		})
	}
}

func TestClonePaneCommandKeepsExitedPaneAndStartsWrapper(t *testing.T) {
	command := clonePaneCommand("%7", "/home/example", map[string]string{"TOKEN": "a b"}, "/tmp/ttyphoon action.sh")
	want := `split-window -v -k -P -F '#{pane_id}' -t %7 -c "#{pane_current_path}" -e "TOKEN=a b" bash "/tmp/ttyphoon action.sh"`
	if command != want {
		t.Fatalf("clonePaneCommand() = %q, want %q", command, want)
	}
}

func TestInteractiveActionCallbackRequiresMatchingUUIDAndRunsOnce(t *testing.T) {
	tmux := &Tmux{panes: newPaneMap()}
	pane := &PaneT{id: "%7", tmux: tmux}
	tmux.panes.Set("%7", pane)
	var calls int
	var gotOutput []byte
	var gotExitCode int
	var gotErr error
	id, err := newInteractiveActionID()
	if err != nil {
		t.Fatal(err)
	}
	tmux.registerInteractiveAction(id, "", func(output []byte, exitCode int, callbackErr error) {
		calls++
		gotOutput = output
		gotExitCode = exitCode
		gotErr = callbackErr
	})
	if tmux.completeInteractiveAction("different-id", "%7", []byte("wrong"), 0) {
		t.Fatal("completion with a different UUID was accepted")
	}
	if !tmux.completeInteractiveAction(id, "%7", []byte("git output"), 1) {
		t.Fatal("early completion was not queued")
	}
	if calls != 0 {
		t.Fatal("callback ran before pane initialization completed")
	}
	if !tmux.bindInteractiveActionPane(id, "%7") {
		t.Fatal("failed to bind callback to initialized pane")
	}
	if tmux.completeInteractiveAction(id, "%8", []byte("wrong pane"), 0) {
		t.Fatal("completion from a different pane was accepted")
	}
	if tmux.completeInteractiveAction(id, "%7", []byte("duplicate"), 0) {
		t.Fatal("duplicate completion was accepted")
	}
	if calls != 1 || string(gotOutput) != "git output" || gotExitCode != 1 || gotErr != nil {
		t.Fatalf("callback result = calls:%d output:%q exit:%d error:%v", calls, gotOutput, gotExitCode, gotErr)
	}
	if pane.InteractiveActionChrome() != types.COLOR_ERROR {
		t.Fatal("failed interactive action did not retain error chrome")
	}
	if pane.AutoCloseAt() != 0 {
		t.Fatal("failed interactive action has an auto-close deadline")
	}
}

func TestInteractiveActionRegistryDoesNotReplaceExistingUUID(t *testing.T) {
	tmux := &Tmux{}
	firstCalls, secondCalls := 0, 0
	if !tmux.registerInteractiveAction("same-id", "", func([]byte, int, error) { firstCalls++ }) {
		t.Fatal("first UUID registration failed")
	}
	if tmux.registerInteractiveAction("same-id", "", func([]byte, int, error) { secondCalls++ }) {
		t.Fatal("duplicate UUID registration unexpectedly succeeded")
	}
	if !tmux.bindInteractiveActionPane("same-id", "%11") {
		t.Fatal("failed to bind original callback to pane")
	}
	if !tmux.completeInteractiveAction("same-id", "%11", nil, 1) {
		t.Fatal("original UUID callback did not complete")
	}
	if firstCalls != 1 || secondCalls != 0 {
		t.Fatalf("callback calls = first:%d second:%d, want first only", firstCalls, secondCalls)
	}
}

func TestSuccessfulInteractiveActionUsesSuccessChrome(t *testing.T) {
	previousDelay := config.Config.Terminal.PaneAutoCloseDelay
	config.Config.Terminal.PaneAutoCloseDelay = "7s"
	t.Cleanup(func() { config.Config.Terminal.PaneAutoCloseDelay = previousDelay })

	tmux := &Tmux{panes: newPaneMap()}
	pane := &PaneT{id: "%10", tmux: tmux}
	tmux.panes.Set("%10", pane)
	t.Cleanup(func() { tmux.panes.Delete(pane.id) })
	id, err := newInteractiveActionID()
	if err != nil {
		t.Fatal(err)
	}
	tmux.registerInteractiveAction(id, "", func([]byte, int, error) {})
	if !tmux.bindInteractiveActionPane(id, pane.id) {
		t.Fatal("failed to bind callback to pane")
	}
	if !tmux.completeInteractiveAction(id, pane.id, []byte("success"), 0) {
		t.Fatal("matching successful completion was not accepted")
	}
	if pane.InteractiveActionChrome() != types.COLOR_OK {
		t.Fatal("successful interactive action did not retain success chrome")
	}
	deadline := pane.AutoCloseAt()
	remaining := time.Until(time.UnixMilli(deadline))
	if remaining < 6*time.Second || remaining > 7*time.Second {
		t.Fatalf("auto-close deadline is %s away, want 7 seconds", remaining)
	}
	if err := tmux.CancelPaneAutoClose(pane.id); err != nil {
		t.Fatal(err)
	}
	if pane.AutoCloseAt() != 0 || pane.autoCloseAt.CompareAndSwap(deadline, -1) {
		t.Fatal("cancelled auto-close can still fire")
	}
	if !pane.CommandFinished() {
		t.Fatal("cancelling auto-close disabled manual close")
	}
}

func TestCancelPaneAutoCloseRejectsAlreadyClosingPane(t *testing.T) {
	tmux := &Tmux{panes: newPaneMap()}
	pane := &PaneT{id: "%10", tmux: tmux}
	tmux.panes.Set(pane.id, pane)
	pane.autoCloseAt.Store(-1)
	if err := tmux.CancelPaneAutoClose(pane.id); err == nil {
		t.Fatal("cancellation was accepted after auto-close started")
	}
	if err := tmux.CancelPaneAutoClose("%missing"); err == nil {
		t.Fatal("cancellation was accepted for a missing pane")
	}
}

func TestClosingInteractiveActionPaneFailsPendingCallback(t *testing.T) {
	tmux := &Tmux{}
	var callbackErr error
	id, err := newInteractiveActionID()
	if err != nil {
		t.Fatal(err)
	}
	tmux.registerInteractiveAction(id, "", func(_ []byte, _ int, err error) {
		callbackErr = err
	})
	if !tmux.bindInteractiveActionPane(id, "%9") {
		t.Fatal("failed to bind callback to pane")
	}

	tmux.cancelInteractiveActionsForPane("%9")
	if callbackErr == nil || callbackErr.Error() != "interactive action pane closed before completion" {
		t.Fatalf("callback error = %v, want a pane-closed error", callbackErr)
	}
	if tmux.completeInteractiveAction(id, "%9", nil, 0) {
		t.Fatal("completion was accepted after pane-close cancellation")
	}
}

func TestWriteBashScriptEmitsCallbackUUIDAndExitCode(t *testing.T) {
	path, err := writeBashScript("git commit -m 'message'", "550e8400-e29b-41d4-a716-446655440000", "/tmp/project")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	script, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"set +e",
		"_ttyphoon_exit_code=$?",
		`ttyphoon;interactive-action-complete;{"id":"%s","exitCode":%s}`,
		"550e8400-e29b-41d4-a716-446655440000",
	} {
		if !strings.Contains(string(script), want) {
			t.Errorf("Bash script does not contain %q", want)
		}
	}
}

func TestBashScriptEmitsCompletionAfterCommandFailure(t *testing.T) {
	const id = "550e8400-e29b-41d4-a716-446655440000"
	path, err := writeBashScript("printf 'hook failed\\n'; false", id, "/tmp/project")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 != 0 {
		t.Fatal("temporary wrapper should be read by Bash, not made executable")
	}
	output, err := exec.Command("bash", path).CombinedOutput()
	if err != nil {
		t.Fatalf("wrapper script exited unexpectedly: %v, output %q", err, output)
	}
	if !strings.Contains(string(output), "hook failed") || !strings.Contains(string(output), `ttyphoon;interactive-action-complete;{"id":"`+id+`","exitCode":1}`) {
		t.Fatalf("wrapper output %q does not include command output and failed completion APC", output)
	}
}

func TestBashScriptReportsEncodedSourcePanePath(t *testing.T) {
	for _, pwd := range []string{
		"/tmp/project",
		"/tmp/space #?%/$HOME/$(printf injected)/quote'\"/caf\u00e9",
		"/tmp/line\nbreak",
	} {
		t.Run(pwd, func(t *testing.T) {
			path, err := writeBashScript(":", "550e8400-e29b-41d4-a716-446655440000", pwd)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Remove(path) })
			command := exec.Command("bash", path)
			command.Dir = t.TempDir()
			command.Env = append(os.Environ(), "HOSTNAME=synthetic-host")
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("wrapper failed: %v, %q", err, output)
			}
			sequence, _, found := strings.Cut(string(output), "\x1b\\")
			if !found || !strings.HasPrefix(sequence, "\x1b]7;file://synthetic-host/") {
				t.Fatalf("missing OSC 7 file URI: %q", output)
			}
			uri, err := url.Parse(strings.TrimPrefix(sequence, "\x1b]7;"))
			if err != nil || uri.Path != pwd || uri.Host != "synthetic-host" || uri.RawQuery != "" || uri.Fragment != "" {
				t.Fatalf("OSC 7 URI = %+v, %v; want path %q", uri, err, pwd)
			}
			if strings.ContainsAny(sequence, "\r\n\t ") {
				t.Fatalf("OSC 7 contains unencoded whitespace: %q", sequence)
			}
		})
	}
}
