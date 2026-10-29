package git

import (
	"errors"
	"fmt"
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
	result := map[string]string{"mode": "", "stdout": "", "stderr": "", "error": ""}
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
		actionResult := <-completed
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
	case "commit":
		return ActionModeInteractive, true
	case "fetch", "fetch-branch", "pull-tags", "push-tag":
		return ActionModeInteractive, true
	default:
		return "", false
	}
}
