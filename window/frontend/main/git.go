package mainwindow

import (
	"context"
	"errors"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/file/dirwatch"
	gitutils "github.com/lmorg/ttyphoon/utils/git"
	"github.com/lmorg/ttyphoon/utils/notes"
	shellutils "github.com/lmorg/ttyphoon/utils/shell"
	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
)

type gitStatusCacheEntry struct {
	status gitutils.Status
	err    string
}

func (a *WApp) GetGitStatus(projectRoot string) (gitutils.Status, error) {
	key, err := gitStatusCacheKey(projectRoot)
	if err != nil {
		return gitutils.Status{}, err
	}
	a.gitStatusMu.Lock()
	entry, ok := a.gitStatusCache[key]
	a.gitStatusMu.Unlock()
	if ok {
		if entry.err != "" {
			return gitutils.Status{}, errors.New(entry.err)
		}
		return cloneGitStatus(entry.status), nil
	}
	return a.refreshGitStatus(projectRoot)
}

func (a *WApp) GetGitBranchDiffFiles(projectRoot, branch string) ([]gitutils.FileStatus, error) {
	return gitutils.BranchDiffFiles(projectRoot, branch)
}

func (a *WApp) GetGitBrowserLink(projectRoot, branch string) (gitutils.BrowserLink, error) {
	return gitutils.RemoteBrowserLink(projectRoot, branch)
}

func (a *WApp) ListGitCommits(projectRoot string, limit int) ([]gitutils.Commit, error) {
	return gitutils.ListCommits(projectRoot, limit)
}

func (a *WApp) ListGitCommitsWithFiles(projectRoot string, limit int) ([]gitutils.Commit, error) {
	return gitutils.ListCommitsWithFiles(projectRoot, limit)
}

func (a *WApp) RefreshGitStatus(projectRoot string) (gitutils.Status, error) {
	return a.refreshGitStatus(projectRoot)
}

func (a *WApp) SetGitStatusWatch(projectRoot string, enabled bool) error {
	a.gitWatchMu.Lock()
	a.gitWatchRequest++
	request := a.gitWatchRequest
	if !enabled {
		a.stopGitStatusWatchLocked()
		a.gitWatchMu.Unlock()
		return nil
	}
	a.gitWatchMu.Unlock()

	key, err := gitStatusCacheKey(projectRoot)
	if err != nil {
		a.gitWatchMu.Lock()
		if request == a.gitWatchRequest {
			a.stopGitStatusWatchLocked()
		}
		a.gitWatchMu.Unlock()
		return err
	}
	directories, err := gitutils.MetadataDirectories(projectRoot)
	if err != nil {
		a.gitWatchMu.Lock()
		if request == a.gitWatchRequest {
			a.stopGitStatusWatchLocked()
		}
		a.gitWatchMu.Unlock()
		return err
	}

	a.gitWatchMu.Lock()
	if request != a.gitWatchRequest {
		a.gitWatchMu.Unlock()
		return nil
	}
	if a.gitWatchRoot == key && a.gitWatchCancel != nil {
		a.gitWatchMu.Unlock()
		return nil
	}
	if a.gitWatchCancel != nil {
		a.gitWatchCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.gitWatchCancel = cancel
	a.gitWatchRoot = key
	a.gitWatchMu.Unlock()

	go a.watchGitStatus(ctx, key, directories)
	return nil
}

func (a *WApp) stopGitStatusWatch() {
	a.gitWatchMu.Lock()
	a.gitWatchRequest++
	a.stopGitStatusWatchLocked()
	a.gitWatchMu.Unlock()
}

func (a *WApp) stopGitStatusWatchLocked() {
	if a.gitWatchCancel != nil {
		a.gitWatchCancel()
	}
	a.gitWatchCancel = nil
	a.gitWatchRoot = ""
}

func (a *WApp) watchGitStatus(ctx context.Context, root string, directories []string) {
	changes := make(chan struct{}, 1)
	ready := make(chan error, len(directories))
	var watchers sync.WaitGroup
	for _, directory := range directories {
		directory := directory
		watchers.Add(1)
		go func() {
			defer watchers.Done()
			dirwatch.RunWithOptions(ctx, directory, func(path string) bool {
				return ignoredGitMetadataPath(directory, path)
			}, dirwatch.Options{WatchFileWrites: true}, func(dirwatch.Change) {
				select {
				case changes <- struct{}{}:
				default:
				}
			}, func(err error) {
				ready <- err
			})
		}()
	}
	defer watchers.Wait()
	for range directories {
		select {
		case err := <-ready:
			if err != nil && ctx.Err() == nil {
				log.Printf("Git metadata watch setup for %q: %v", root, err)
			}
		case <-ctx.Done():
			return
		}
	}

	_, _ = a.refreshGitStatus(root)
	var timer *time.Timer
	var timerChannel <-chan time.Time
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-changes:
			if timer == nil {
				timer = time.NewTimer(250 * time.Millisecond)
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(250 * time.Millisecond)
			}
			timerChannel = timer.C
		case <-timerChannel:
			timerChannel = nil
			_, _ = a.refreshGitStatus(root)
		}
	}
}

func ignoredGitMetadataPath(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	first := strings.SplitN(relative, string(filepath.Separator), 2)[0]
	return first == "objects" || first == "hooks" || first == "logs"
}

func (a *WApp) refreshGitStatus(projectRoot string) (gitutils.Status, error) {
	key, err := gitStatusCacheKey(projectRoot)
	if err != nil {
		return gitutils.Status{}, err
	}
	a.gitStatusMu.Lock()
	if a.gitStatusCache == nil {
		a.gitStatusCache = map[string]gitStatusCacheEntry{}
	}
	if a.gitStatusRequest == nil {
		a.gitStatusRequest = map[string]uint64{}
	}
	a.gitStatusRequest[key]++
	request := a.gitStatusRequest[key]
	a.gitStatusMu.Unlock()

	status, statusErr := gitutils.WorkspaceStatus(projectRoot)
	entry := gitStatusCacheEntry{status: cloneGitStatus(status)}
	if statusErr != nil {
		entry.err = statusErr.Error()
	}
	a.gitStatusMu.Lock()
	if a.gitStatusRequest[key] == request {
		if statusErr == nil {
			a.gitStatusCache[key] = entry
		} else {
			delete(a.gitStatusCache, key)
		}
	}
	a.gitStatusMu.Unlock()

	if a.ctx != nil {
		update := map[string]any{"root": projectRoot, "status": status}
		if statusErr != nil {
			update["status"] = nil
			update["error"] = statusErr.Error()
		}
		runtime.EventsEmit(a.ctx, "notesGitStatusChanged", update)
	}
	return status, statusErr
}

func (a *WApp) refreshGitStatusForFileIndex(root string, event notes.FileIndexEvent) {
	if event.State == "scanning" || event.State == "refreshing" {
		return
	}
	contextRoot, err := gitStatusCacheKey(a.currentNotesContext().PathProjectRoot)
	if err != nil {
		return
	}
	eventRoot, err := gitStatusCacheKey(root)
	if err != nil || eventRoot != contextRoot {
		return
	}
	go func() {
		_, _ = a.refreshGitStatus(root)
	}()
}

func gitStatusCacheKey(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", errors.New("Git workspace root is empty")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if resolved, resolveErr := filepath.EvalSymlinks(absolute); resolveErr == nil {
		absolute = resolved
	}
	return filepath.Clean(absolute), nil
}

func cloneGitStatus(status gitutils.Status) gitutils.Status {
	status.Files = append([]gitutils.FileStatus(nil), status.Files...)
	return status
}

func (a *WApp) GetGitDiff(projectRoot, filePath, originalPath string) (gitutils.Diff, error) {
	diff, err := gitutils.ReadDiff(projectRoot, filePath, originalPath)
	if err != nil {
		return gitutils.Diff{}, err
	}
	return diff, nil
}

func (a *WApp) GetGitDiffAgainstBranch(projectRoot, filePath, originalPath, branch string) (gitutils.Diff, error) {
	diff, err := gitutils.ReadDiff(projectRoot, filePath, originalPath, branch)
	if err != nil {
		return gitutils.Diff{}, err
	}
	return diff, nil
}

func (a *WApp) StageGitFile(projectRoot, filePath string) error {
	return gitutils.StageFile(projectRoot, filePath)
}

func (a *WApp) StageAllGitFiles(projectRoot string) error {
	return gitutils.StageAllTrackedFiles(projectRoot)
}

func (a *WApp) RunGitAction(projectRoot, actionName string, parameters map[string]string) map[string]string {
	result := gitutils.RunAction(projectRoot, gitutils.ActionRequest{
		Name:       actionName,
		Parameters: parameters,
	}, func(command string, complete func(gitutils.InteractiveActionResult)) error {
		renderer, ok := renderwebkit.CurrentRenderer()
		if !ok {
			return errors.New("Notes renderer is unavailable")
		}
		return renderer.ClonePane(command, func(output []byte, exitCode int, err error) {
			complete(gitutils.InteractiveActionResult{
				Output:   string(output),
				ExitCode: exitCode,
				Err:      err,
			})
		})
	})
	if result["error"] == "" {
		switch strings.ToLower(strings.TrimSpace(actionName)) {
		case "fetch", "fetch-branch", "pull-tags", "push":
			_, _ = a.refreshGitStatus(projectRoot)
		}
	}
	return result
}

func (a *WApp) ConfirmGitPush(projectRoot string) map[string]string {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return map[string]string{"error": "Notes renderer is unavailable"}
	}
	decision := make(chan *types.InputBoxCallbackResultT, 1)
	renderer.DisplayInputBoxW(&types.InputBoxWT{
		Options: types.InputBoxWTOptions{
			Title:           "Confirm Git push",
			Message:         "Run git push using the repository's configured remote and push settings?",
			Confirmation:    true,
			ConfirmLabel:    "Push",
			CancelLabel:     "Cancel",
			ConfirmIsDanger: true,
		},
		OkFunc: func(value *types.InputBoxCallbackResultT) {
			decision <- value
		},
		CancelFunc: func(*types.InputBoxCallbackResultT) { decision <- nil },
	})
	var confirmed *types.InputBoxCallbackResultT
	if a.ctx != nil {
		select {
		case confirmed = <-decision:
		case <-a.ctx.Done():
			return map[string]string{"error": "Git push confirmation was cancelled"}
		}
	} else {
		confirmed = <-decision
	}
	if confirmed == nil {
		return map[string]string{"cancelled": "true"}
	}
	return a.RunGitAction(projectRoot, "push", nil)
}

func (a *WApp) UnstageGitFile(projectRoot, filePath string) error {
	return gitutils.UnstageFile(projectRoot, filePath)
}

func (a *WApp) ListGitWorktrees(projectRoot string) ([]gitutils.Worktree, error) {
	return gitutils.ListWorktrees(projectRoot)
}

func (a *WApp) ListGitBranches(projectRoot string) ([]string, error) {
	return gitutils.ListBranches(projectRoot)
}

func (a *WApp) ListGitTags(projectRoot string) ([]string, error) {
	return gitutils.ListTags(projectRoot)
}

func (a *WApp) CreateGitTag(projectRoot, tag string) error {
	return gitutils.CreateTag(projectRoot, tag)
}

func (a *WApp) DeleteGitTag(projectRoot, tag string) error {
	return gitutils.DeleteTag(projectRoot, tag)
}

func (a *WApp) CheckoutGitTag(projectRoot, tag string) error {
	if err := gitutils.CheckoutTag(projectRoot, tag); err != nil {
		return err
	}
	_, _ = a.refreshGitStatus(projectRoot)
	return nil
}

func (a *WApp) CreateGitTagWorktree(projectRoot, tag, worktreePath string) (gitutils.Worktree, error) {
	worktree, err := gitutils.CreateTagWorktree(projectRoot, tag, worktreePath)
	if err != nil {
		return gitutils.Worktree{}, err
	}
	_, _ = a.refreshGitStatus(projectRoot)
	return worktree, nil
}

func (a *WApp) PullGitTags(projectRoot string) (map[string]string, error) {
	return a.RunGitAction(projectRoot, "pull-tags", nil), nil
}

func (a *WApp) PushGitTag(projectRoot, tag string) (map[string]string, error) {
	return a.RunGitAction(projectRoot, "push-tag", map[string]string{"tag": tag}), nil
}

func (a *WApp) CheckoutGitBranch(projectRoot, branch string, create bool) error {
	if err := gitutils.CheckoutBranch(projectRoot, branch, create); err != nil {
		return err
	}
	_, _ = a.refreshGitStatus(projectRoot)
	return nil
}

func (a *WApp) DeleteGitBranch(projectRoot, branch string) error {
	if err := gitutils.DeleteBranch(projectRoot, branch); err != nil {
		return err
	}
	_, _ = a.refreshGitStatus(projectRoot)
	return nil
}

func (a *WApp) FetchGitBranch(projectRoot, branch string) (map[string]string, error) {
	return a.RunGitAction(projectRoot, "fetch-branch", map[string]string{"branch": branch}), nil
}

func (a *WApp) GetDefaultGitWorktreePath(projectRoot string) (string, error) {
	return gitutils.DefaultWorktreePath(projectRoot)
}

func (a *WApp) CreateGitWorktree(projectRoot, branch, worktreePath string, createBranch bool) (gitutils.Worktree, error) {
	worktree, err := gitutils.CreateWorktree(projectRoot, branch, worktreePath, createBranch)
	if err != nil {
		return gitutils.Worktree{}, err
	}
	_, _ = a.refreshGitStatus(projectRoot)
	return worktree, nil
}

func (a *WApp) SwitchGitWorktree(projectRoot, worktreePath string) error {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return errors.New("Notes renderer is unavailable")
	}
	worktrees, err := gitutils.ListWorktrees(projectRoot)
	if err != nil {
		return err
	}
	selectedPath, err := canonicalGitPath(worktreePath)
	if err != nil {
		return err
	}
	for _, worktree := range worktrees {
		listedPath, pathErr := canonicalGitPath(worktree.Path)
		if pathErr == nil && listedPath == selectedPath {
			tile := renderer.ActiveTile()
			if tile == nil || tile.GetTerm() == nil {
				return errors.New("active terminal is unavailable")
			}
			shellutils.UpdateReadline(tile.GetTerm(), "cd "+quoteShellArgument(listedPath))
			return nil
		}
	}
	return errors.New("selected path is not a registered worktree for the current repository")
}

func (a *WApp) RemoveGitWorktree(projectRoot, worktreePath string) error {
	if err := gitutils.RemoveWorktree(projectRoot, worktreePath); err != nil {
		return err
	}
	_, err := a.refreshGitStatus(projectRoot)
	return err
}

func (a *WApp) ShowGitWorktreeInfo(projectRoot, worktreePath string) error {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return errors.New("Notes renderer is unavailable")
	}
	return showGitInfoModel(renderer, "git-info:"+worktreePath, func() (string, error) {
		return gitutils.RenderWorktreeInfo(projectRoot, worktreePath)
	})
}

func (a *WApp) ShowGitRefInfo(projectRoot, name string, isTag bool) error {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return errors.New("Notes renderer is unavailable")
	}
	kind := "branch"
	if isTag {
		kind = "tag"
	}
	return showGitInfoModel(renderer, "git-info:"+kind+":"+name, func() (string, error) {
		return gitutils.RenderGitRefInfo(projectRoot, name, isTag)
	})
}

func showGitInfoModel(renderer types.Renderer, id string, generate func() (string, error)) error {
	sticky := renderer.DisplaySticky(types.NOTIFY_INFO, "Generating diffs", func() {})
	markdown, err := generate()
	if err != nil {
		sticky.Close()
		return err
	}
	sticky.Close()
	renderer.DisplayMarkdownModel(id, markdown)
	return nil
}

func quoteShellArgument(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func canonicalGitPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}
