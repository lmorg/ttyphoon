package git

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type ActionMode string

const (
	ActionModeGo          ActionMode = "go"
	ActionModeInteractive ActionMode = "interactive"
)

type InteractiveActionResult struct {
	Output   string
	ExitCode int
	Err      error
}

type InteractiveActionRunner func(command string, complete func(InteractiveActionResult)) error

type ActionRequest struct {
	Name       string
	Parameters map[string]string
}

func RunAction(root string, action ActionRequest, agentEnvironment map[string]string, interactiveRunner InteractiveActionRunner) map[string]string {
	return RunActionContext(context.Background(), root, action, agentEnvironment, interactiveRunner)
}

func RunActionContext(ctx context.Context, root string, action ActionRequest, agentEnvironment map[string]string, interactiveRunner InteractiveActionRunner) map[string]string {
	result := map[string]string{"mode": "", "stdout": "", "stderr": "", "error": ""}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		result["error"] = err.Error()
		return result
	}
	mode, ok := actionMode(action.Name)
	if !ok {
		result["error"] = fmt.Sprintf("unsupported Git action %q", action.Name)
		return result
	}
	result["mode"] = string(mode)

	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		result["error"] = err.Error()
		return result
	}

	args, err := actionArguments(gitPath, repoRoot, action)
	if err != nil {
		result["error"] = err.Error()
		return result
	}
	for _, argument := range args {
		if strings.ContainsAny(argument, "\x00\r\n") {
			result["error"] = "Git action argument contains unsupported control characters"
			return result
		}
	}
	if mode == ActionModeInteractive {
		if interactiveRunner == nil {
			result["error"] = errors.New("interactive Git action runner is unavailable").Error()
			return result
		}
		completed := make(chan InteractiveActionResult, 1)
		if err := interactiveRunner(interactiveCommand(repoRoot, args), func(actionResult InteractiveActionResult) {
			select {
			case completed <- actionResult:
			default:
			}
		}); err != nil {
			result["error"] = err.Error()
			return result
		}
		var actionResult InteractiveActionResult
		select {
		case actionResult = <-completed:
		case <-ctx.Done():
			result["error"] = ctx.Err().Error()
			return result
		}
		result["stdout"] = actionResult.Output
		result["exitCode"] = fmt.Sprintf("%d", actionResult.ExitCode)
		if actionResult.Err != nil {
			result["error"] = actionResult.Err.Error()
		} else if actionResult.ExitCode != 0 {
			result["error"] = fmt.Sprintf("git action failed with exit code %d", actionResult.ExitCode)
		}
		return result
	}

	output, err := runGitWithAgentEnvironment(gitPath, repoRoot, agentEnvironment, args...)
	if err != nil {
		result["error"] = err.Error()
		return result
	}
	for key, value := range output {
		result[key] = value
	}
	return result
}

func interactiveCommand(projectRoot string, args []string) string {
	parts := make([]string, 0, len(args)+3)
	parts = append(parts, "git", "-C", quoteShellArgument(projectRoot))
	for _, argument := range args {
		parts = append(parts, quoteShellArgument(argument))
	}
	return strings.Join(parts, " ")
}

func quoteShellArgument(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func actionArguments(gitPath, repoRoot string, action ActionRequest) ([]string, error) {
	switch strings.ToLower(strings.TrimSpace(action.Name)) {
	case "commit":
		message := strings.TrimSpace(action.Parameters["message"])
		if message == "" {
			return nil, errors.New("commit message is required")
		}
		return []string{"commit", "-m", message}, nil
	case "push":
		return []string{"push"}, nil
	case "status":
		return []string{"--no-pager", "status", "--short", "--branch"}, nil
	case "diff":
		args := []string{"--no-pager", "diff", "--no-ext-diff", "--no-color", "--unified=3"}
		if action.Parameters["staged"] != "" {
			staged, err := strconv.ParseBool(action.Parameters["staged"])
			if err != nil {
				return nil, errors.New("staged must be true or false")
			}
			if staged {
				args = append(args, "--cached")
			}
		}
		if path := strings.TrimSpace(action.Parameters["path"]); path != "" {
			path, err := validateRelativePath(path)
			if err != nil {
				return nil, err
			}
			args = append(args, "--", filepath.ToSlash(path))
		}
		return args, nil
	case "log":
		limit := 10
		if rawLimit := strings.TrimSpace(action.Parameters["limit"]); rawLimit != "" {
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil || parsed < 1 || parsed > 50 {
				return nil, errors.New("log limit must be between 1 and 50")
			}
			limit = parsed
		}
		revision, err := checkedCommitRevision(gitPath, repoRoot, action.Parameters["revision"])
		if err != nil {
			return nil, err
		}
		return []string{"--no-pager", "log", "--oneline", "--decorate", "--no-color", "--max-count", strconv.Itoa(limit), revision}, nil
	case "show":
		revision, err := checkedCommitRevision(gitPath, repoRoot, action.Parameters["revision"])
		if err != nil {
			return nil, err
		}
		args := []string{"--no-pager", "show", "--no-ext-diff", "--no-color", "--format=fuller", "--stat", "--no-renames", revision}
		if path := strings.TrimSpace(action.Parameters["path"]); path != "" {
			path, err := validateRelativePath(path)
			if err != nil {
				return nil, err
			}
			args = append(args, "--", filepath.ToSlash(path))
		}
		return args, nil
	case "branches":
		return []string{"--no-pager", "branch", "--all", "--no-color"}, nil
	case "remotes":
		return []string{"remote"}, nil
	case "fetch":
		return []string{"fetch"}, nil
	case "fetch-branch":
		return branchFetchArguments(gitPath, repoRoot, action.Parameters["branch"])
	case "pull-tags":
		return []string{"fetch", "origin", "refs/tags/*:refs/tags/*"}, nil
	case "push-tag":
		ref, err := checkedLocalTag(gitPath, repoRoot, action.Parameters["tag"])
		if err != nil {
			return nil, err
		}
		return []string{"push", "origin", ref}, nil
	default:
		return nil, fmt.Errorf("unsupported Git action %q", action.Name)
	}
}

func checkedCommitRevision(gitPath, repoRoot, revision string) (string, error) {
	revision = strings.TrimSpace(revision)
	if revision == "" {
		revision = "HEAD"
	}
	if strings.ContainsAny(revision, "\x00\r\n") {
		return "", errors.New("revision contains unsupported control characters")
	}
	output, err := runGit(gitPath, repoRoot, "rev-parse", "--verify", "--quiet", "--end-of-options", revision+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("invalid Git revision %q", revision)
	}
	return strings.TrimSpace(string(output)), nil
}

func branchFetchArguments(gitPath, repoRoot, branch string) ([]string, error) {
	ref, err := checkedLocalBranch(gitPath, repoRoot, branch)
	if err != nil {
		return nil, err
	}
	upstream, err := runGit(gitPath, repoRoot, "for-each-ref", "--format=%(upstream:remotename)%00%(upstream:remoteref)", ref)
	if err != nil {
		return nil, err
	}
	fields := strings.Split(strings.TrimRight(string(upstream), "\r\n"), "\x00")
	remote, remoteRef := "origin", ref
	if len(fields) == 2 && fields[0] != "" && fields[1] != "" {
		remote, remoteRef = fields[0], fields[1]
	}
	destination := "refs/remotes/" + remote + "/" + strings.TrimPrefix(remoteRef, "refs/heads/")
	if remote == "." {
		destination = remoteRef
	}
	return []string{"fetch", "--", remote, remoteRef + ":" + destination}, nil
}

func actionMode(action string) (ActionMode, bool) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "commit", "push":
		return ActionModeInteractive, true
	case "fetch", "fetch-branch", "pull-tags", "push-tag", "status", "diff", "log", "show", "branches", "remotes":
		return ActionModeInteractive, true
	default:
		return "", false
	}
}
