package git

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func initGitTestRepo(t *testing.T) string {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	runGitTestCommand(t, gitPath, root, "init", "--quiet", "--initial-branch=main")
	runGitTestCommand(t, gitPath, root, "config", "user.name", "Test User")
	runGitTestCommand(t, gitPath, root, "config", "user.email", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(root, "tracked file.txt"), []byte("head contents\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, gitPath, root, "add", "--", "tracked file.txt")
	runGitTestCommand(t, gitPath, root, "commit", "--quiet", "-m", "initial")
	return root
}

func runGitTestCommand(t *testing.T, gitPath, root string, args ...string) {
	t.Helper()
	commandArgs := append([]string{"-C", root}, args...)
	cmd := exec.Command(gitPath, commandArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func TestGitTransferLogsOutputByCommandOutcome(t *testing.T) {
	for _, test := range []struct {
		name string
		exit string
	}{
		{"success", "0"},
		{"failure", "1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			gitPath := filepath.Join(root, "git")
			script := "#!/bin/sh\nprintf '%s\\n' 'stdout one' 'stdout two'\nprintf '%s\\n' 'stderr one' 'stderr two' >&2\nexit " + test.exit + "\n"
			if err := os.WriteFile(gitPath, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			previous := log.Writer()
			log.SetOutput(&output)
			t.Cleanup(func() { log.SetOutput(previous) })
			result, err := runGitWithOutput(gitPath, root, "fetch")
			if err != nil {
				t.Fatal(err)
			}
			if (result["error"] != "") != (test.exit != "0") {
				t.Fatalf("command error = %q for exit %s", result["error"], test.exit)
			}
			for _, line := range []string{"stdout one", "stdout two"} {
				if strings.Contains(output.String(), "[info] "+line) != (test.exit == "0") {
					t.Errorf("stdout info classification incorrect for %s: %s", test.name, line)
				}
			}
			for _, line := range []string{"stderr one", "stderr two"} {
				if strings.Contains(output.String(), "[info] "+line) != (test.exit == "0") {
					t.Errorf("stderr info classification incorrect for %s: %s", test.name, line)
				}
			}
			if result["stdout"] != "stdout one\nstdout two" || result["stderr"] != "stderr one\nstderr two" {
				t.Fatalf("command output changed: %v", result)
			}
		})
	}
}

func TestGitCommandEnvironmentInheritsProcessEnvironment(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "/app/agent.sock")
	t.Setenv("SSH_AGENT_PID", "11")
	t.Setenv("GIT_TERMINAL_PROMPT", "1")
	values := make(map[string]string)
	counts := make(map[string]int)
	for _, entry := range gitCommandEnvironment() {
		name, value, _ := strings.Cut(entry, "=")
		values[name] = value
		counts[name]++
	}
	if values["SSH_AUTH_SOCK"] != "/app/agent.sock" || values["SSH_AGENT_PID"] != "11" {
		t.Fatal("Git environment did not inherit the process SSH agent")
	}
	if values["GIT_TERMINAL_PROMPT"] != "0" || counts["GIT_TERMINAL_PROMPT"] != 1 {
		t.Fatal("interactive Git prompting was not disabled exactly once")
	}
}

func TestFetchBranchInheritsProcessEnvironment(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("transport fixture uses a POSIX wrapper")
	}
	root := initGitTestRepo(t)
	remote := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, gitPath, root, "checkout", "--quiet", "-b", "agent-test")
	runGitTestCommand(t, gitPath, remote, "checkout", "--quiet", "-b", "agent-test")
	runGitTestCommand(t, gitPath, root, "remote", "add", "origin", remote)
	directory := t.TempDir()
	capture := filepath.Join(directory, "agent-environment")
	wrapper := filepath.Join(directory, "upload-pack")
	t.Setenv("FETCH_TEST_CAPTURE", capture)
	t.Setenv("FETCH_TEST_GIT", gitPath)
	t.Setenv("SSH_AUTH_SOCK", "/app/agent.sock")
	t.Setenv("SSH_AGENT_PID", "11")
	script := "#!/bin/sh\nprintf '%s\\n%s\\n' \"$SSH_AUTH_SOCK\" \"$SSH_AGENT_PID\" > \"$FETCH_TEST_CAPTURE\"\nexec \"$FETCH_TEST_GIT\" upload-pack \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, gitPath, root, "config", "remote.origin.uploadpack", wrapper)
	result, err := FetchBranch(root, "agent-test")
	if err != nil || result["error"] != "" {
		t.Fatalf("fetch = %v, %v", result, err)
	}
	actual, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != "/app/agent.sock\n11\n" {
		t.Fatalf("transport agent environment = %q", actual)
	}
}

func TestBranchCheckoutAndFetch(t *testing.T) {
	root := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckoutBranch(root, "topic/new", true); err != nil {
		t.Fatal(err)
	}
	status, err := WorkspaceStatus(root)
	if err != nil || status.Branch != "topic/new" {
		t.Fatalf("created branch = %+v, %v", status, err)
	}
	remote := initGitTestRepo(t)
	runGitTestCommand(t, gitPath, remote, "checkout", "--quiet", "-b", "topic/new")
	runGitTestCommand(t, gitPath, remote, "commit", "--quiet", "--allow-empty", "-m", "remote update")
	runGitTestCommand(t, gitPath, root, "remote", "add", "origin", remote)
	head, err := exec.Command(gitPath, "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	result, err := FetchBranch(root, "topic/new")
	if err != nil || result["error"] != "" {
		t.Fatalf("fetch = %v, %v", result, err)
	}
	after, err := exec.Command(gitPath, "-C", root, "rev-parse", "HEAD").Output()
	if err != nil || string(head) != string(after) {
		t.Fatalf("fetch changed HEAD: %s, %v", after, err)
	}
	remoteHead, err := exec.Command(gitPath, "-C", remote, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	fetched, err := exec.Command(gitPath, "-C", root, "rev-parse", "refs/remotes/origin/topic/new").Output()
	if err != nil || string(fetched) != string(remoteHead) {
		t.Fatalf("fetched tip = %s, %v", fetched, err)
	}
	runGitTestCommand(t, gitPath, remote, "checkout", "--quiet", "-b", "remote/topic")
	runGitTestCommand(t, gitPath, remote, "commit", "--quiet", "--allow-empty", "-m", "upstream update")
	runGitTestCommand(t, gitPath, root, "config", "branch.topic/new.remote", "origin")
	runGitTestCommand(t, gitPath, root, "config", "branch.topic/new.merge", "refs/heads/remote/topic")
	result, err = FetchBranch(root, "topic/new")
	if err != nil || result["error"] != "" {
		t.Fatalf("upstream fetch = %v, %v", result, err)
	}
	remoteHead, err = exec.Command(gitPath, "-C", remote, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	fetched, err = exec.Command(gitPath, "-C", root, "rev-parse", "refs/remotes/origin/remote/topic").Output()
	if err != nil || string(fetched) != string(remoteHead) {
		t.Fatalf("upstream tip = %s, %v", fetched, err)
	}
	if err := CheckoutBranch(root, "other", true); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tracked file.txt"), []byte("other branch\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, gitPath, root, "add", "tracked file.txt")
	runGitTestCommand(t, gitPath, root, "commit", "--quiet", "-m", "other content")
	if err := CheckoutBranch(root, "topic/new", false); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tracked file.txt"), []byte("unsaved edits\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CheckoutBranch(root, "other", false); err == nil {
		t.Fatal("checkout overwrote dirty content")
	}
	status, err = WorkspaceStatus(root)
	if err != nil || status.Branch != "topic/new" {
		t.Fatalf("failed checkout changed branch: %+v, %v", status, err)
	}
	runGitTestCommand(t, gitPath, root, "config", "branch.topic/new.merge", "refs/heads/missing")
	result, err = FetchBranch(root, "topic/new")
	if err != nil || result["error"] == "" || result["stderr"] == "" {
		t.Fatalf("fetch failure output = %v, %v", result, err)
	}
	for _, invalid := range []string{"invalid branch", "--detach", "HEAD~1"} {
		if err := CheckoutBranch(root, invalid, false); err == nil {
			t.Fatalf("accepted %q", invalid)
		}
	}
}

func TestTagListCheckoutWorktreeAndDelete(t *testing.T) {
	root := initGitTestRepo(t)
	for _, tag := range []string{"v5.9.9", "v1.0.0", "v10.3.4-rc.1", "v10.3.4", "release"} {
		if err := CreateTag(root, tag); err != nil {
			t.Fatalf("CreateTag(%q) = %v", tag, err)
		}
	}
	tags, err := ListTags(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"v10.3.4", "v10.3.4-rc.1", "v5.9.9", "v1.0.0", "release"}
	if strings.Join(tags, ",") != strings.Join(want, ",") {
		t.Fatalf("ListTags() = %v, want %v", tags, want)
	}
	if err := CheckoutTag(root, "v5.9.9"); err != nil {
		t.Fatalf("CheckoutTag() = %v", err)
	}
	status, err := WorkspaceStatus(root)
	if err != nil || status.Branch != "(detached HEAD)" {
		t.Fatalf("tag checkout status = %+v, %v", status, err)
	}
	worktreePath := filepath.Join(t.TempDir(), "v5.9.9")
	worktree, err := CreateTagWorktree(root, "v5.9.9", worktreePath)
	if err != nil || worktree.Branch != "tag: v5.9.9" {
		t.Fatalf("CreateTagWorktree() = %+v, %v", worktree, err)
	}
	worktrees, err := ListWorktrees(root)
	if err != nil {
		t.Fatal(err)
	}
	foundTagWorktree := false
	canonicalWorktreePath, err := filepath.EvalSymlinks(worktreePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range worktrees {
		listedPath, pathErr := filepath.EvalSymlinks(item.Path)
		if pathErr == nil && listedPath == canonicalWorktreePath && strings.HasPrefix(item.Branch, "tag: ") {
			foundTagWorktree = true
		}
	}
	if !foundTagWorktree {
		t.Fatalf("tag worktree is not labeled as a tag: %+v", worktrees)
	}
	if err := DeleteTag(root, "v10.3.4-rc.1"); err != nil {
		t.Fatalf("DeleteTag() = %v", err)
	}
	if err := DeleteTag(root, "missing"); err == nil {
		t.Fatal("DeleteTag accepted a missing tag")
	}
	if err := CreateTag(root, "invalid tag"); err == nil {
		t.Fatal("CreateTag accepted an invalid name")
	}
}

func TestPullAndPushTagsInheritProcessEnvironment(t *testing.T) {
	root := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	remote := filepath.Join(t.TempDir(), "remote.git")
	if err := os.Mkdir(remote, 0700); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, gitPath, remote, "init", "--quiet", "--bare")
	runGitTestCommand(t, gitPath, root, "remote", "add", "origin", remote)
	if err := CreateTag(root, "v1.2.3"); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, gitPath, root, "push", "origin", "refs/tags/v1.2.3")
	if err := DeleteTag(root, "v1.2.3"); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	capture := filepath.Join(directory, "agent-environment")
	t.Setenv("FETCH_TEST_CAPTURE", capture)
	t.Setenv("FETCH_TEST_GIT", gitPath)
	t.Setenv("SSH_AUTH_SOCK", "/app/agent.sock")
	t.Setenv("SSH_AGENT_PID", "11")
	script := "#!/bin/sh\nprintf '%s\\n%s\\n' \"$SSH_AUTH_SOCK\" \"$SSH_AGENT_PID\" > \"$FETCH_TEST_CAPTURE\"\nexec \"$FETCH_TEST_GIT\" $FETCH_TEST_SERVICE \"$@\"\n"
	wrapper := filepath.Join(directory, "git-transport")
	if err := os.WriteFile(wrapper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FETCH_TEST_SERVICE", "upload-pack")
	runGitTestCommand(t, gitPath, root, "config", "remote.origin.uploadpack", wrapper)
	result, err := PullTags(root)
	if err != nil || result["error"] != "" {
		t.Fatalf("PullTags() = %v, %v", result, err)
	}
	actual, err := os.ReadFile(capture)
	if err != nil || string(actual) != "/app/agent.sock\n11\n" {
		t.Fatalf("pull transport environment = %q, %v", actual, err)
	}
	if tags, err := ListTags(root); err != nil || len(tags) != 1 || tags[0] != "v1.2.3" {
		t.Fatalf("tags after PullTags() = %v, %v", tags, err)
	}
	runGitTestCommand(t, gitPath, remote, "tag", "--delete", "v1.2.3")
	t.Setenv("FETCH_TEST_SERVICE", "receive-pack")
	runGitTestCommand(t, gitPath, root, "config", "remote.origin.receivepack", wrapper)
	result, err = PushTag(root, "v1.2.3")
	if err != nil || result["error"] != "" {
		t.Fatalf("PushTag() = %v, %v", result, err)
	}
	actual, err = os.ReadFile(capture)
	if err != nil || string(actual) != "/app/agent.sock\n11\n" {
		t.Fatalf("push transport environment = %q, %v", actual, err)
	}
	if _, err := exec.Command(gitPath, "--git-dir", remote, "rev-parse", "--verify", "refs/tags/v1.2.3").Output(); err != nil {
		t.Fatalf("remote tag missing after PushTag(): %v", err)
	}
}

func TestDeleteBranchRemovesOnlySafeLocalBranches(t *testing.T) {
	root := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(gitPath, "-C", root, "branch", "--show-current").Output()
	if err != nil {
		t.Fatal(err)
	}
	current := strings.TrimSpace(string(output))
	if err := CheckoutBranch(root, "topic/delete-me", true); err != nil {
		t.Fatal(err)
	}
	if err := CheckoutBranch(root, current, false); err != nil {
		t.Fatal(err)
	}
	if err := DeleteBranch(root, current); err == nil {
		t.Fatalf("deleted checked-out branch %q", current)
	}
	if err := DeleteBranch(root, "topic/delete-me"); err != nil {
		t.Fatalf("DeleteBranch() = %v", err)
	}
	branches, err := ListBranches(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, branch := range branches {
		if branch == "topic/delete-me" {
			t.Fatal("deleted branch is still listed")
		}
	}
	if err := DeleteBranch(root, "--force"); err == nil {
		t.Fatal("DeleteBranch accepted an invalid branch name")
	}
}

func TestListCommitsWithFilesIncludesChangesAndRenamePaths(t *testing.T) {
	root := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	oldName := "old file\nname.txt"
	newName := "nested/new file.txt"
	if err := os.WriteFile(filepath.Join(root, oldName), []byte("content\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, gitPath, root, "add", "--", oldName)
	runGitTestCommand(t, gitPath, root, "commit", "--quiet", "-m", "add document")
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, gitPath, root, "mv", "--", oldName, newName)
	runGitTestCommand(t, gitPath, root, "commit", "--quiet", "-m", "rename document")
	commits, err := ListCommitsWithFiles(root, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 3 || !slices.Contains(commits[0].Files, oldName) || !slices.Contains(commits[0].Files, newName) || !slices.Contains(commits[1].Files, oldName) {
		t.Fatalf("changed paths = %+v", commits)
	}
	if !slices.Contains(commits[2].Files, "tracked file.txt") {
		t.Fatalf("root commit paths = %v", commits[2].Files)
	}
	runGitTestCommand(t, gitPath, root, "rm", "--", newName)
	runGitTestCommand(t, gitPath, root, "commit", "--quiet", "-m", "delete document")
	deleted, err := ListCommitsWithFiles(root, 1)
	if err != nil || len(deleted) != 1 || !slices.Contains(deleted[0].Files, newName) {
		t.Fatalf("deleted paths = %+v, %v", deleted, err)
	}
	plain, err := ListCommits(root, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, commit := range plain {
		if len(commit.Files) != 0 {
			t.Fatalf("default history unexpectedly loads changed paths: %+v", commit)
		}
	}
}

func TestListCommitsIncludesMetadataAndGraph(t *testing.T) {
	root := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, gitPath, root, "commit", "--amend", "--quiet", "-m", "initial", "-m", "Body-only search phrase")
	runGitTestCommand(t, gitPath, root, "tag", "release,stable")
	runGitTestCommand(t, gitPath, root, "tag", "-a", "v1.0.0", "-m", "Release")
	runGitTestCommand(t, gitPath, root, "branch", "not-a-tag")
	commits, err := ListCommits(root, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 1 || commits[0].Subject != "initial" || commits[0].Author != "Test User" {
		t.Fatalf("ListCommits() = %+v", commits)
	}
	if len(commits[0].Hash) < 7 || len(commits[0].Date) != 19 || !strings.Contains(commits[0].Graph[0], "*") {
		t.Fatalf("commit metadata/graph = %+v", commits[0])
	}
	fullHash, err := exec.Command(gitPath, "-C", root, "rev-parse", "HEAD").Output()
	if err != nil || commits[0].FullHash != strings.TrimSpace(string(fullHash)) {
		t.Fatalf("full commit hash = %q, %v", commits[0].FullHash, err)
	}
	if commits[0].Message != "initial\n\nBody-only search phrase" {
		t.Fatalf("commit message = %q", commits[0].Message)
	}
	if got := strings.Join(commits[0].Tags, "|"); got != "release,stable|v1.0.0" {
		t.Fatalf("commit tags = %q, want lightweight and annotated tags only", got)
	}
	runGitTestCommand(t, gitPath, root, "commit", "--allow-empty", "--quiet", "-m", "untagged")
	commits, err = ListCommits(root, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 || len(commits[0].Tags) != 0 || len(commits[1].Tags) != 2 {
		t.Fatalf("tags do not belong to the correct commits: %+v", commits)
	}
	if _, err := ListCommits(root, 0); err == nil {
		t.Fatal("ListCommits accepted an invalid limit")
	}
}

func TestListCommitsPreservesMergeGraphAndLimit(t *testing.T) {
	root := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	output, err := exec.Command(gitPath, "-C", root, "branch", "--show-current").Output()
	if err != nil {
		t.Fatal(err)
	}
	branch := strings.TrimSpace(string(output))
	runGitTestCommand(t, gitPath, root, "checkout", "--quiet", "-b", "feature")
	if err := os.WriteFile(filepath.Join(root, "feature.txt"), []byte("feature\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, gitPath, root, "add", "feature.txt")
	runGitTestCommand(t, gitPath, root, "commit", "--quiet", "--allow-empty", "-m", "Feature commit")
	runGitTestCommand(t, gitPath, root, "checkout", "--quiet", branch)
	runGitTestCommand(t, gitPath, root, "commit", "--quiet", "--allow-empty", "-m", "Main commit")
	runGitTestCommand(t, gitPath, root, "merge", "--quiet", "--no-ff", "feature", "-m", "Merge feature")
	commits, err := ListCommits(root, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 4 || commits[0].Subject != "Merge feature" {
		t.Fatalf("merge history = %+v", commits)
	}
	var graph strings.Builder
	for _, commit := range commits {
		graph.WriteString(strings.Join(commit.Graph, "\n"))
	}
	if !strings.Contains(graph.String(), "\\") || !strings.Contains(graph.String(), "/") {
		t.Fatalf("merge connectors missing from graph: %q", graph.String())
	}
	limited, err := ListCommits(root, 2)
	if err != nil || len(limited) != 2 {
		t.Fatalf("limited history = %+v, %v", limited, err)
	}
	withFiles, err := ListCommitsWithFiles(root, 2)
	if err != nil || len(withFiles) != 2 || !slices.Contains(withFiles[0].Files, "feature.txt") {
		t.Fatalf("merge paths with limit = %+v, %v", withFiles, err)
	}
}

func TestListCommitsScopesHistoryToHead(t *testing.T) {
	root := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	output, err := exec.Command(gitPath, "-C", root, "branch", "--show-current").Output()
	if err != nil {
		t.Fatal(err)
	}
	branch := strings.TrimSpace(string(output))
	runGitTestCommand(t, gitPath, root, "checkout", "--quiet", "-b", "unmerged")
	runGitTestCommand(t, gitPath, root, "commit", "--quiet", "--allow-empty", "-m", "Unmerged commit")
	runGitTestCommand(t, gitPath, root, "tag", "unmerged-tag")
	runGitTestCommand(t, gitPath, root, "update-ref", "refs/remotes/origin/unmerged", "HEAD")
	runGitTestCommand(t, gitPath, root, "checkout", "--quiet", branch)

	commits, err := ListCommits(root, 100)
	if err != nil || len(commits) != 1 || commits[0].Subject != "initial" {
		t.Fatalf("current branch history = %+v, %v; want only initial commit", commits, err)
	}
	runGitTestCommand(t, gitPath, root, "checkout", "--quiet", "--detach", "unmerged")
	commits, err = ListCommits(root, 100)
	if err != nil || len(commits) != 2 || commits[0].Subject != "Unmerged commit" {
		t.Fatalf("detached HEAD history = %+v, %v; want detached commit's ancestry", commits, err)
	}
	runGitTestCommand(t, gitPath, root, "checkout", "--quiet", "--orphan", "unborn")
	commits, err = ListCommits(root, 100)
	if err != nil || len(commits) != 0 {
		t.Fatalf("unborn branch history = %+v, %v; want no commits", commits, err)
	}
}

func TestWorkspaceStatusDoesNotRewriteIndex(t *testing.T) {
	root := initGitTestRepo(t)
	t.Setenv("GIT_OPTIONAL_LOCKS", "1")
	indexPath := filepath.Join(root, ".git", "index")
	before, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeInfo, err := os.Stat(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	trackedPath := filepath.Join(root, "tracked file.txt")
	stale := time.Now().Add(-time.Hour)
	if err := os.Chtimes(trackedPath, stale, stale); err != nil {
		t.Fatal(err)
	}
	status, err := WorkspaceStatus(root)
	if err != nil || !status.Clean {
		t.Fatalf("timestamp-only status = %+v, %v; want clean", status, err)
	}
	if err := os.WriteFile(trackedPath, []byte("changed contents\n"), 0600); err != nil {
		t.Fatal(err)
	}
	status, err = WorkspaceStatus(root)
	if err != nil || status.Clean || !filesForPath(status.Files, "tracked file.txt").Unstaged {
		t.Fatalf("changed-content status = %+v, %v; want an unstaged change", status, err)
	}
	after, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	afterInfo, err := os.Stat(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || !beforeInfo.ModTime().Equal(afterInfo.ModTime()) {
		t.Fatal("status refreshed the index on disk; this can retrigger the metadata watch")
	}
}

func TestWorkspaceStatusAndStageLifecycle(t *testing.T) {
	root := initGitTestRepo(t)
	if err := os.WriteFile(filepath.Join(root, "tracked file.txt"), []byte("working contents\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "new file.txt"), []byte("new contents\n"), 0600); err != nil {
		t.Fatal(err)
	}

	status, err := WorkspaceStatus(root)
	if err != nil {
		t.Fatalf("WorkspaceStatus() error = %v", err)
	}
	if status.Root != resolveTestPath(t, root) || status.Clean || len(status.Files) != 2 {
		t.Fatalf("WorkspaceStatus() = %+v, want root, dirty state, and two files", status)
	}
	files := make(map[string]FileStatus, len(status.Files))
	for _, file := range status.Files {
		files[file.Path] = file
	}
	if !files["tracked file.txt"].Unstaged || files["tracked file.txt"].Untracked {
		t.Fatalf("tracked file status = %+v, want unstaged tracked change", files["tracked file.txt"])
	}
	if !files["new file.txt"].Untracked || files["new file.txt"].Staged {
		t.Fatalf("new file status = %+v, want untracked change", files["new file.txt"])
	}

	diff, err := ReadDiff(root, "tracked file.txt")
	if err != nil {
		t.Fatalf("ReadDiff(tracked) error = %v", err)
	}
	if diff.Original != "head contents\n" || diff.Working != "working contents\n" {
		t.Fatalf("ReadDiff(tracked) = %+v", diff)
	}

	diff, err = ReadDiff(root, "new file.txt")
	if err != nil {
		t.Fatalf("ReadDiff(untracked) error = %v", err)
	}
	if diff.Original != "" || diff.Working != "new contents\n" {
		t.Fatalf("ReadDiff(untracked) = %+v, want empty original and working contents", diff)
	}

	if err := StageFile(root, "new file.txt"); err != nil {
		t.Fatalf("StageFile() error = %v", err)
	}
	status, err = WorkspaceStatus(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Files) != 2 || !status.Files[0].Staged && !status.Files[1].Staged {
		t.Fatalf("staged status = %+v, want staged file", status.Files)
	}

	if err := UnstageFile(root, "new file.txt"); err != nil {
		t.Fatalf("UnstageFile() error = %v", err)
	}
	status, err = WorkspaceStatus(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Files) != 2 || !filesForPath(status.Files, "new file.txt").Untracked {
		t.Fatalf("unstaged status = %+v, want new file restored to untracked", status.Files)
	}
}

func TestStageAllTrackedFilesDoesNotStageUntracked(t *testing.T) {
	root := initGitTestRepo(t)
	if err := os.WriteFile(filepath.Join(root, "tracked file.txt"), []byte("changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "new file.txt"), []byte("untracked\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := StageAllTrackedFiles(root); err != nil {
		t.Fatalf("StageAllTrackedFiles() error = %v", err)
	}
	status, err := WorkspaceStatus(root)
	if err != nil {
		t.Fatal(err)
	}
	tracked := filesForPath(status.Files, "tracked file.txt")
	untracked := filesForPath(status.Files, "new file.txt")
	if !tracked.Staged || tracked.Untracked {
		t.Errorf("tracked file status = %+v, want staged", tracked)
	}
	if !untracked.Untracked || untracked.Staged {
		t.Errorf("untracked file status = %+v, want unstaged untracked", untracked)
	}
}

func TestRunActionUsesInteractivePolicyForCommit(t *testing.T) {
	root := initGitTestRepo(t)
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	var called bool
	result := RunAction(root, ActionRequest{Name: "commit", Parameters: map[string]string{"message": "Update"}}, func(command string, complete func(InteractiveActionResult)) error {
		called = true
		want := "git -C '" + canonicalRoot + "' 'commit' '-m' 'Update'"
		if command != want {
			t.Errorf("interactive command = %q, want %q", command, want)
		}
		complete(InteractiveActionResult{Output: "commit output", ExitCode: 0})
		return nil
	})
	if result["mode"] != string(ActionModeInteractive) || result["error"] != "" || result["stdout"] != "commit output" || result["exitCode"] != "0" || !called {
		t.Fatalf("RunAction() = %v, runner called = %v; want successful interactive dispatch", result, called)
	}
}

func TestRunActionReturnsInteractiveFailureOutputAndExitCode(t *testing.T) {
	root := initGitTestRepo(t)
	result := RunAction(root, ActionRequest{Name: "commit", Parameters: map[string]string{"message": "Update"}}, func(_ string, complete func(InteractiveActionResult)) error {
		complete(InteractiveActionResult{Output: "hook rejected commit", ExitCode: 1})
		return nil
	})
	if result["stdout"] != "hook rejected commit" || result["exitCode"] != "1" || result["error"] != "git action failed with exit code 1" {
		t.Fatalf("RunAction() = %v, want interactive output and failure status", result)
	}
}

func TestInteractiveCommandQuotesRootAndArguments(t *testing.T) {
	command := interactiveCommand("/tmp/repo's dir", []string{"commit", "-m", "rock'n'roll"})
	const want = `git -C '/tmp/repo'\''s dir' 'commit' '-m' 'rock'\''n'\''roll'`
	if command != want {
		t.Fatalf("interactiveCommand() = %q, want %q", command, want)
	}
}

func TestRunActionUsesInteractivePolicyForRemoteOperations(t *testing.T) {
	root := initGitTestRepo(t)
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, mustLookPathGit(t), root, "tag", "v1.2.3")
	for _, test := range []struct {
		name       string
		parameters map[string]string
		wantArgs   []string
	}{
		{name: "fetch", wantArgs: []string{"fetch"}},
		{name: "fetch-branch", parameters: map[string]string{"branch": "main"}, wantArgs: []string{"fetch", "--", "origin", "refs/heads/main:refs/remotes/origin/main"}},
		{name: "pull-tags", wantArgs: []string{"fetch", "origin", "refs/tags/*:refs/tags/*"}},
		{name: "push-tag", parameters: map[string]string{"tag": "v1.2.3"}, wantArgs: []string{"push", "origin", "refs/tags/v1.2.3"}},
		{name: "push", wantArgs: []string{"push"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var called bool
			result := RunAction(root, ActionRequest{Name: test.name, Parameters: test.parameters}, func(command string, complete func(InteractiveActionResult)) error {
				called = true
				want := interactiveCommand(canonicalRoot, test.wantArgs)
				if command != want {
					t.Errorf("interactive command = %q, want %q", command, want)
				}
				complete(InteractiveActionResult{Output: "transfer output", ExitCode: 0})
				return nil
			})
			if result["mode"] != string(ActionModeInteractive) || result["error"] != "" || !called {
				t.Fatalf("RunAction() = %v, interactive runner called = %v; want interactive execution", result, called)
			}
		})
	}
}

func TestRunActionReadOnlyCommandsUseInteractiveRunner(t *testing.T) {
	root := initGitTestRepo(t)
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	gitPath := mustLookPathGit(t)
	head, err := runGit(gitPath, root, "rev-parse", "--verify", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	headHash := strings.TrimSpace(string(head))

	tests := []struct {
		name       string
		parameters map[string]string
		wantArgs   []string
	}{
		{name: "status", wantArgs: []string{"--no-pager", "status", "--short", "--branch"}},
		{name: "diff", parameters: map[string]string{"path": "tracked file.txt"}, wantArgs: []string{"--no-pager", "diff", "--no-ext-diff", "--no-color", "--unified=3", "--", "tracked file.txt"}},
		{name: "log", parameters: map[string]string{"limit": "5", "revision": "HEAD"}, wantArgs: []string{"--no-pager", "log", "--oneline", "--decorate", "--no-color", "--max-count", "5", headHash}},
		{name: "show", parameters: map[string]string{"revision": "HEAD", "path": "tracked file.txt"}, wantArgs: []string{"--no-pager", "show", "--no-ext-diff", "--no-color", "--format=fuller", "--stat", "--no-renames", headHash, "--", "tracked file.txt"}},
		{name: "branches", wantArgs: []string{"--no-pager", "branch", "--all", "--no-color"}},
		{name: "remotes", wantArgs: []string{"remote"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			result := RunAction(root, ActionRequest{Name: test.name, Parameters: test.parameters}, func(command string, complete func(InteractiveActionResult)) error {
				called = true
				if want := interactiveCommand(canonicalRoot, test.wantArgs); command != want {
					t.Errorf("interactive command = %q, want %q", command, want)
				}
				complete(InteractiveActionResult{Output: "read-only result", ExitCode: 0})
				return nil
			})
			if !called || result["mode"] != string(ActionModeInteractive) || result["error"] != "" {
				t.Fatalf("RunAction() = %v, runner called = %v; want interactive read", result, called)
			}
		})
	}
}

func TestRunActionRejectsUnsafeReadInputsAndMutatingActions(t *testing.T) {
	root := initGitTestRepo(t)
	called := false
	runner := func(string, func(InteractiveActionResult)) error {
		called = true
		return nil
	}
	for _, test := range []struct {
		name       string
		parameters map[string]string
	}{
		{name: "reset"},
		{name: "diff", parameters: map[string]string{"path": "../outside.txt"}},
		{name: "log", parameters: map[string]string{"limit": "1000"}},
	} {
		result := RunAction(root, ActionRequest{Name: test.name, Parameters: test.parameters}, runner)
		if result["error"] == "" {
			t.Errorf("RunAction(%q) unexpectedly succeeded", test.name)
		}
	}
	if called {
		t.Fatal("interactive runner was called for an invalid or mutating action")
	}
}

func TestRunActionContextCanStopWaitingForClonePane(t *testing.T) {
	root := initGitTestRepo(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := RunActionContext(ctx, root, ActionRequest{Name: "status"}, func(string, func(InteractiveActionResult)) error {
		cancel()
		return nil
	})
	if !strings.Contains(result["error"], context.Canceled.Error()) {
		t.Fatalf("RunActionContext() error = %q, want cancellation", result["error"])
	}
}

func mustLookPathGit(t *testing.T) string {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	return gitPath
}

func TestMetadataDirectoriesIncludeLinkedWorktreeCommonDirectory(t *testing.T) {
	root := initGitTestRepo(t)
	worktree := filepath.Join(t.TempDir(), "linked-worktree")
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	runGitTestCommand(t, gitPath, root, "worktree", "add", "--quiet", "--detach", worktree, "HEAD")

	directories, err := MetadataDirectories(worktree)
	if err != nil {
		t.Fatalf("MetadataDirectories() error = %v", err)
	}
	if len(directories) != 2 {
		t.Fatalf("MetadataDirectories() = %v, want per-worktree and common directories", directories)
	}
	for _, directory := range directories {
		if directory == filepath.Join(worktree, ".git") {
			t.Fatalf("MetadataDirectories() returned the .git file path: %v", directories)
		}
		if _, err := os.Stat(directory); err != nil {
			t.Fatalf("metadata directory %q is not accessible: %v", directory, err)
		}
	}
}

func TestBrowserLinkForRemote(t *testing.T) {
	tests := []struct {
		name     string
		remote   string
		branch   string
		provider string
		url      string
	}{
		{"GitHub HTTPS", "https://github.com/org/repo.git", "feature/view", "github", "https://github.com/org/repo/tree/feature/view"},
		{"GitHub SSH", "git@github.com:org/repo.git", "main", "github", "https://github.com/org/repo/tree/main"},
		{"GitLab nested group", "git@gitlab.com:org/group/repo.git", "feature/view", "gitlab", "https://gitlab.com/org/group/repo/-/tree/feature/view"},
		{"Bitbucket", "git@bitbucket.org:org/repo.git", "main", "bitbucket", "https://bitbucket.org/org/repo/src/main/"},
		{"Azure HTTPS", "https://dev.azure.com/org/project/_git/repo", "feature/view", "azuredevops", "https://dev.azure.com/org/project/_git/repo?version=GBfeature%2Fview"},
		{"Azure SSH", "git@ssh.dev.azure.com:v3/org/project/repo", "main", "azuredevops", "https://dev.azure.com/org/project/_git/repo?version=GBmain"},
		{"Azure legacy host", "https://org.visualstudio.com/project/_git/repo", "main", "azuredevops", "https://dev.azure.com/org/project/_git/repo?version=GBmain"},
		{"unsupported host", "git@example.com:org/repo.git", "main", "", ""},
		{"local path", "../repo", "main", "", ""},
		{"detached head", "https://github.com/org/repo.git", "HEAD", "", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			link := browserLinkForRemote(test.remote, test.branch)
			if link.Provider != test.provider || link.URL != test.url {
				t.Fatalf("browserLinkForRemote(%q, %q) = %+v, want provider %q and URL %q", test.remote, test.branch, link, test.provider, test.url)
			}
		})
	}
}

func TestRemoteBrowserLinkReadsOriginFromRepository(t *testing.T) {
	root := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	link, err := RemoteBrowserLink(root, "main")
	if err != nil || link.URL != "" {
		t.Fatalf("RemoteBrowserLink() without origin = %+v, %v; want no link", link, err)
	}
	runGitTestCommand(t, gitPath, root, "remote", "add", "origin", "git@github.com:org/repo.git")

	link, err = RemoteBrowserLink(root, "feature/ui")
	if err != nil {
		t.Fatalf("RemoteBrowserLink() error = %v", err)
	}
	if link.Provider != "github" || link.URL != "https://github.com/org/repo/tree/feature/ui" {
		t.Fatalf("RemoteBrowserLink() = %+v, want GitHub feature branch URL", link)
	}
}

func TestBranchDiffFilesAndReadDiffDoNotSwitchBranches(t *testing.T) {
	root := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	branchCommand := exec.Command(gitPath, "-C", root, "branch", "--show-current")
	branchOutput, err := branchCommand.Output()
	if err != nil {
		t.Fatal(err)
	}
	currentBranch := strings.TrimSpace(string(branchOutput))
	unchangedPath := "unchanged file\nname.txt"
	if err := os.WriteFile(filepath.Join(root, unchangedPath), []byte("same contents\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, gitPath, root, "add", "--", unchangedPath)
	runGitTestCommand(t, gitPath, root, "commit", "--quiet", "-m", "unchanged file")
	runGitTestCommand(t, gitPath, root, "checkout", "--quiet", "-b", "compare-base")
	if err := os.WriteFile(filepath.Join(root, "tracked file.txt"), []byte("compare contents\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "branch-only.txt"), []byte("branch file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, gitPath, root, "add", "--", "tracked file.txt", "branch-only.txt")
	runGitTestCommand(t, gitPath, root, "commit", "--quiet", "-m", "comparison base")
	runGitTestCommand(t, gitPath, root, "checkout", "--quiet", currentBranch)
	if err := os.WriteFile(filepath.Join(root, "untracked.txt"), []byte("local file\n"), 0600); err != nil {
		t.Fatal(err)
	}

	files, err := BranchDiffFiles(root, "compare-base")
	if err != nil {
		t.Fatalf("BranchDiffFiles() error = %v", err)
	}
	if len(files) != 4 {
		t.Fatalf("BranchDiffFiles() = %+v, want tracked, branch-only, untracked, and equal paths", files)
	}
	if last := files[len(files)-1]; last.Path != unchangedPath || last.Code != "=" || last.Staged || last.Unstaged || last.Untracked {
		t.Fatalf("last comparison entry = %+v, want unchanged file without staging actions", last)
	}
	for _, file := range files {
		if !file.CompareOnly {
			t.Errorf("file %q CompareOnly = false", file.Path)
		}
	}
	diff, err := ReadDiff(root, "tracked file.txt", "", "compare-base")
	if err != nil {
		t.Fatalf("ReadDiff(comparison branch) error = %v", err)
	}
	if diff.Original != "compare contents\n" || diff.Working != "head contents\n" {
		t.Fatalf("ReadDiff(comparison branch) = %+v, want comparison base and current working contents", diff)
	}
	status, err := WorkspaceStatus(root)
	if err != nil {
		t.Fatal(err)
	}
	if status.Branch != currentBranch {
		t.Fatalf("WorkspaceStatus().Branch = %q, want unchanged branch %q", status.Branch, currentBranch)
	}
}

func TestCommitComparisonUsesCommitWithoutChangingHead(t *testing.T) {
	root := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	original, err := exec.Command(gitPath, "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	commit := strings.TrimSpace(string(original))
	if err := os.WriteFile(filepath.Join(root, "tracked file.txt"), []byte("later contents\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, gitPath, root, "add", "--", "tracked file.txt")
	runGitTestCommand(t, gitPath, root, "commit", "--quiet", "-m", "later")
	head, err := exec.Command(gitPath, "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{commit, commit[:7]} {
		files, err := BranchDiffFiles(root, target)
		if err != nil || len(files) != 1 {
			t.Fatalf("commit comparison files = %+v, %v", files, err)
		}
		diff, err := ReadDiff(root, "tracked file.txt", "", target)
		if err != nil || diff.Original != "head contents\n" || diff.Working != "later contents\n" {
			t.Fatalf("commit diff = %+v, %v", diff, err)
		}
	}
	after, err := exec.Command(gitPath, "-C", root, "rev-parse", "HEAD").Output()
	if err != nil || string(after) != string(head) {
		t.Fatalf("HEAD changed after comparison: %s, %v", after, err)
	}
	runGitTestCommand(t, gitPath, root, "branch", commit[:8], "HEAD")
	files, err := BranchDiffFiles(root, commit[:8])
	if err != nil || len(files) != 1 || files[0].Code != "=" || !files[0].CompareOnly {
		t.Fatalf("hex-named branch precedence = %+v, %v", files, err)
	}
	for _, target := range []string{"HEAD~1", "--all", "0000000"} {
		if _, err := BranchDiffFiles(root, target); err == nil {
			t.Fatalf("accepted invalid comparison target %q", target)
		}
	}
}

func TestWorkspaceStatusAcceptsWorktreeRoot(t *testing.T) {
	root := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	worktree := filepath.Join(t.TempDir(), "linked-worktree")
	runGitTestCommand(t, gitPath, root, "worktree", "add", "--quiet", "-b", "linked", worktree)

	status, err := WorkspaceStatus(worktree)
	if err != nil {
		t.Fatalf("WorkspaceStatus(worktree) error = %v", err)
	}
	if status.Root != resolveTestPath(t, worktree) || status.Branch != "linked" {
		t.Fatalf("WorkspaceStatus(worktree) = %+v, want worktree root and linked branch", status)
	}
	info, err := os.Stat(filepath.Join(worktree, ".git"))
	if err != nil || info.IsDir() {
		t.Fatalf("worktree .git entry = %v, %v; want a gitdir file", info, err)
	}
}

func TestListBranchesAndWorktreesFromEitherRootAndCreateWorktrees(t *testing.T) {
	base := resolveTestPath(t, t.TempDir())
	root := filepath.Join(base, "primary repo")
	linkedRoot := filepath.Join(base, " linked worktree")
	binDir := filepath.Join(base, "bin")
	for _, path := range []string{filepath.Join(root, ".git", "info"), linkedRoot, binDir} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(linkedRoot, ".git"), []byte("gitdir: synthetic\n"), 0600); err != nil {
		t.Fatal(err)
	}
	commandLog := filepath.Join(base, "worktree-commands")
	t.Setenv("TTYPHOON_TEST_GIT_ROOT", root)
	t.Setenv("TTYPHOON_TEST_GIT_LINKED_ROOT", linkedRoot)
	t.Setenv("TTYPHOON_TEST_GIT_COMMAND_LOG", commandLog)
	t.Setenv("PATH", binDir)
	script := `#!/bin/sh
if [ "$1" != "-C" ]; then
    printf 'expected -C, got %s\n' "$*" >&2
    exit 1
fi
repo=$2
shift 2
case "$repo" in
    "$TTYPHOON_TEST_GIT_ROOT"|"$TTYPHOON_TEST_GIT_LINKED_ROOT") ;;
    *) printf 'unexpected repository: %s\n' "$repo" >&2; exit 1 ;;
esac
if [ "$1" = "worktree" ] && [ "$2" = "add" ]; then
    printf '%s\000' -C "$repo" "$@" >> "$TTYPHOON_TEST_GIT_COMMAND_LOG"
    exit 0
fi
case "$*" in
    'rev-parse --show-toplevel')
        printf '%s\n' "$repo" ;;
    'for-each-ref --format=%(refname:short) refs/heads')
        printf '%s\n' zeta main alpha ;;
    'worktree list --porcelain -z')
        printf 'worktree %s\000HEAD aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\000branch refs/heads/main\000\000' "$TTYPHOON_TEST_GIT_ROOT"
        printf 'worktree %s\000HEAD aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\000branch refs/heads/zeta\000\000' "$TTYPHOON_TEST_GIT_LINKED_ROOT" ;;
    'check-ref-format refs/heads/alpha'|'check-ref-format refs/heads/feature/new') ;;
    'rev-parse --git-common-dir')
        printf '%s/.git\n' "$TTYPHOON_TEST_GIT_ROOT" ;;
    '--no-optional-locks status --porcelain=v1 -z --branch --untracked-files=all')
        printf '## main\000' ;;
    *) printf 'unexpected git command: %s\n' "$*" >&2; exit 1 ;;
esac
`
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}

	for _, queryRoot := range []string{root, linkedRoot} {
		branches, err := ListBranches(queryRoot)
		if err != nil {
			t.Fatalf("ListBranches(%q) error = %v", queryRoot, err)
		}
		if len(branches) != 3 || branches[0] != "alpha" || branches[1] != "main" || branches[2] != "zeta" {
			t.Fatalf("ListBranches(%q) = %v, want sorted local branches", queryRoot, branches)
		}
		worktrees, err := ListWorktrees(queryRoot)
		if err != nil {
			t.Fatalf("ListWorktrees(%q) error = %v", queryRoot, err)
		}
		if len(worktrees) != 2 || worktrees[0] != (Worktree{Path: linkedRoot, Branch: "zeta"}) || worktrees[1] != (Worktree{Path: root, Branch: "main"}) {
			t.Fatalf("ListWorktrees(%q) = %+v, want sorted synthetic worktrees with exact paths", queryRoot, worktrees)
		}
	}

	createdPath := filepath.Join(resolveTestPath(t, root), ".git_worktrees", "alpha")
	worktree, err := CreateWorktree(root, "alpha", filepath.Join(".git_worktrees", "alpha"), false)
	if err != nil {
		t.Fatalf("CreateWorktree(existing branch) error = %v", err)
	}
	if worktree.Path != createdPath || worktree.Branch != "alpha" {
		t.Fatalf("CreateWorktree(existing branch) = %+v", worktree)
	}
	status, err := WorkspaceStatus(root)
	if err != nil {
		t.Fatalf("WorkspaceStatus() after nested worktree creation error = %v", err)
	}
	if !status.Clean {
		t.Fatalf("WorkspaceStatus() after nested worktree creation = %+v, want clean parent", status)
	}

	newBranchPath := filepath.Join(".git_worktrees", "feature", "new")
	worktree, err = CreateWorktree(linkedRoot, "feature/new", newBranchPath, true)
	if err != nil {
		t.Fatalf("CreateWorktree(new branch from linked root) error = %v", err)
	}
	if worktree.Branch != "feature/new" || worktree.Path != filepath.Join(linkedRoot, newBranchPath) {
		t.Fatalf("CreateWorktree(new branch) = %+v", worktree)
	}
	commands, err := os.ReadFile(commandLog)
	if err != nil {
		t.Fatal(err)
	}
	wantCommands := []string{
		"-C", root, "worktree", "add", createdPath, "alpha",
		"-C", linkedRoot, "worktree", "add", "-b", "feature/new", filepath.Join(linkedRoot, newBranchPath),
	}
	if string(commands) != strings.Join(wantCommands, "\x00")+"\x00" {
		t.Fatalf("worktree commands = %q, want %q", strings.Split(string(commands), "\x00"), wantCommands)
	}
	exclude, err := os.ReadFile(filepath.Join(root, ".git", "info", "exclude"))
	if err != nil {
		t.Fatal(err)
	}
	if string(exclude) != "/.git_worktrees/alpha/\n/.git_worktrees/feature/new/\n" {
		t.Fatalf("worktree exclusions = %q, want both nested worktree paths in the shared Git directory", exclude)
	}
}

func containsWorktreePath(worktrees []Worktree, path string) bool {
	for _, worktree := range worktrees {
		if worktree.Path == path {
			return true
		}
	}
	return false
}

func TestCreateWorktreeAtCommitCreatesDetachedHistory(t *testing.T) {
	root := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(gitPath, "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	hash := strings.TrimSpace(string(output))
	shortOutput, err := exec.Command(gitPath, "-C", root, "rev-parse", "--short", hash).Output()
	if err != nil {
		t.Fatal(err)
	}
	shortHash := strings.TrimSpace(string(shortOutput))
	runGitTestCommand(t, gitPath, root, "commit", "--quiet", "--allow-empty", "-m", "later")
	target := filepath.Join(t.TempDir(), shortHash)
	worktree, err := CreateWorktree(root, hash, target, false)
	if err != nil {
		t.Fatal(err)
	}
	if worktree.Branch != shortHash {
		t.Fatalf("worktree = %+v", worktree)
	}
	for _, listingRoot := range []string{root, target} {
		listed, err := ListWorktrees(listingRoot)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, entry := range listed {
			if entry.Path == worktree.Path {
				found = true
				if entry.Branch != shortHash {
					t.Fatalf("detached menu label = %q, want %q", entry.Branch, shortHash)
				}
			}
		}
		if !found {
			t.Fatal("detached worktree missing from listing")
		}
	}
	actual, err := exec.Command(gitPath, "-C", target, "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(actual)) != hash {
		t.Fatalf("worktree HEAD = %s, %v", actual, err)
	}
	if err := exec.Command(gitPath, "-C", target, "symbolic-ref", "--quiet", "HEAD").Run(); err == nil {
		t.Fatal("commit worktree is not detached")
	}
	if _, err := CreateWorktree(root, strings.Repeat("0", 40), filepath.Join(t.TempDir(), "invalid"), false); err == nil {
		t.Fatal("accepted a nonexistent commit")
	}
	runGitTestCommand(t, gitPath, root, "branch", "abcdef1")
	hexTarget := filepath.Join(t.TempDir(), "hex-branch")
	if _, err := CreateWorktree(root, "abcdef1", hexTarget, false); err != nil {
		t.Fatal(err)
	}
	branch, err := exec.Command(gitPath, "-C", hexTarget, "symbolic-ref", "--short", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(branch)) != "abcdef1" {
		t.Fatalf("hex branch worktree = %s, %v", branch, err)
	}
}

func TestCreateWorktreeRejectsUnsafePathsAndBranches(t *testing.T) {
	root := initGitTestRepo(t)
	if _, err := CreateWorktree(root, "main", "../outside", false); err == nil {
		t.Fatal("CreateWorktree() error = nil, want path traversal rejection")
	}
	if _, err := CreateWorktree(root, "invalid branch", "safe/path", true); err == nil {
		t.Fatal("CreateWorktree() error = nil, want invalid branch rejection")
	}
}

func TestDefaultWorktreePathUsesOriginOrProjectRootHash(t *testing.T) {
	root := initGitTestRepo(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	rootHash := sha256.Sum256([]byte(resolveTestPath(t, root)))
	path, err := DefaultWorktreePath(root)
	if err != nil {
		t.Fatalf("DefaultWorktreePath() without origin error = %v", err)
	}
	want := filepath.Join(home, "Documents", "ttyphoon", ".git_worktrees", hex.EncodeToString(rootHash[:]))
	if path != want {
		t.Fatalf("DefaultWorktreePath() without origin = %q, want %q", path, want)
	}

	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	origin := "https://example.invalid/team/project.git"
	runGitTestCommand(t, gitPath, root, "remote", "add", "origin", origin)
	originHash := sha256.Sum256([]byte(origin))
	path, err = DefaultWorktreePath(root)
	if err != nil {
		t.Fatalf("DefaultWorktreePath() with origin error = %v", err)
	}
	want = filepath.Join(home, "Documents", "ttyphoon", ".git_worktrees", hex.EncodeToString(originHash[:]))
	if path != want {
		t.Fatalf("DefaultWorktreePath() with origin = %q, want %q", path, want)
	}
}

func TestCreateWorktreeAtAbsolutePathOutsideRepository(t *testing.T) {
	root := initGitTestRepo(t)
	worktreePath := filepath.Join(t.TempDir(), "external-worktree")
	worktree, err := CreateWorktree(root, "external", worktreePath, true)
	if err != nil {
		t.Fatalf("CreateWorktree(absolute path) error = %v", err)
	}
	if worktree.Path != resolveTestPath(t, worktreePath) || worktree.Branch != "external" {
		t.Fatalf("CreateWorktree(absolute path) = %+v, want path %q and branch external", worktree, worktreePath)
	}
}

func TestRemoveWorktreePreservesBranch(t *testing.T) {
	root := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	worktreePath := filepath.Join(t.TempDir(), "removable-worktree")
	runGitTestCommand(t, gitPath, root, "worktree", "add", "--quiet", "-b", "removable", worktreePath)
	if err := RemoveWorktree(root, worktreePath); err != nil {
		t.Fatalf("RemoveWorktree() error = %v", err)
	}
	if _, err := os.Stat(worktreePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("worktree path stat error = %v, want not-exist", err)
	}
	output, err := exec.Command(gitPath, "-C", root, "branch", "--list", "removable").CombinedOutput()
	if err != nil {
		t.Fatalf("git branch --list error = %v: %s", err, output)
	}
	if strings.TrimSpace(string(output)) != "removable" {
		t.Fatalf("branch list after RemoveWorktree = %q, want retained branch removable", output)
	}
}

func TestRenderWorktreeInfoKeepsMultilineDiffInOneTableRow(t *testing.T) {
	root := initGitTestRepo(t)
	if err := os.WriteFile(filepath.Join(root, "tracked file.txt"), []byte("head contents\nnew | <line>\n"), 0600); err != nil {
		t.Fatal(err)
	}
	markdown, err := RenderWorktreeInfo(root, root)
	if err != nil {
		t.Fatalf("RenderWorktreeInfo() error = %v", err)
	}
	if !strings.Contains(markdown, "| <details><summary>tracked file\\.txt</summary><pre><code>diff --git a/tracked file.txt b/tracked file.txt&#10;") {
		t.Fatalf("RenderWorktreeInfo() missing diff table row:\n%s", markdown)
	}
	if !strings.Contains(markdown, "&#10;+new &#124; &lt;line&gt;</code></pre></details> |") {
		t.Fatalf("RenderWorktreeInfo() did not encode multiline diff:\n%s", markdown)
	}
	for _, line := range strings.Split(markdown, "\n") {
		if strings.HasPrefix(line, "| <details><summary>tracked file\\.txt</summary>") && !strings.HasSuffix(line, "</code></pre></details> |") {
			t.Fatalf("diff row was split across lines: %q", line)
		}
	}
	if !strings.Contains(markdown, "git version") {
		t.Fatalf("RenderWorktreeInfo() missing Git version:\n%s", markdown)
	}
}

func TestGitRefInfoShowsCommitChangesAndMatchingTagsWithoutStatus(t *testing.T) {
	root := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	branchOutput, err := exec.Command(gitPath, "-C", root, "branch", "--show-current").Output()
	if err != nil {
		t.Fatal(err)
	}
	rootBranch := strings.TrimSpace(string(branchOutput))
	runGitTestCommand(t, gitPath, root, "checkout", "--quiet", "-b", "release/2.0")
	if err := os.WriteFile(filepath.Join(root, "tracked file.txt"), []byte("committed change\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "added file.txt"), []byte("new file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, gitPath, root, "add", "--", "tracked file.txt", "added file.txt")
	runGitTestCommand(t, gitPath, root, "commit", "--quiet", "-m", "release commit")
	runGitTestCommand(t, gitPath, root, "tag", "-a", "v2.0.0", "-m", "release tag")
	runGitTestCommand(t, gitPath, root, "tag", "v2.0.0-alias")

	info, err := GitRefInfo(root, "release/2.0", false)
	if err != nil {
		t.Fatalf("GitRefInfo(branch) = %v", err)
	}
	if info.IsWorktree || info.RepoStatus != "" || info.BranchName != "release/2.0" {
		t.Fatalf("branch Info fields = %+v", info)
	}
	if len(info.CommitHashLong) != 40 || len(info.MatchedTags) != 2 {
		t.Fatalf("branch commit/tags = %+v", info)
	}
	if len(info.RepoDiffFiles) != 2 || info.RepoDiffFiles[0].Filename != "added file.txt" || info.RepoDiffFiles[1].Filename != "tracked file.txt" {
		t.Fatalf("branch commit diff files = %+v", info.RepoDiffFiles)
	}
	if !strings.Contains(info.RepoDiffFiles[1].Diff, "+committed change") {
		t.Fatalf("commit diff omitted changed contents: %+v", info.RepoDiffFiles[1])
	}
	for _, hash := range []string{info.CommitHashLong, info.CommitHashLong[:8]} {
		commitInfo, err := GitRefInfo(root, hash, false)
		if err != nil {
			t.Fatalf("GitRefInfo(%q) = %v", hash, err)
		}
		if commitInfo.CommitHashLong != info.CommitHashLong || commitInfo.BranchName != "" || commitInfo.IsWorktree || commitInfo.RepoStatus != "" {
			t.Fatalf("commit Info fields = %+v", commitInfo)
		}
		if len(commitInfo.RepoDiffFiles) != 2 || len(commitInfo.MatchedTags) != 2 || !strings.Contains(commitInfo.RepoDiffFiles[1].Diff, "+committed change") {
			t.Fatalf("commit Info omitted changes or tags: %+v", commitInfo)
		}
		markdown, err := RenderGitRefInfo(root, hash, false)
		if err != nil {
			t.Fatalf("RenderGitRefInfo(%q) = %v", hash, err)
		}
		if !strings.Contains(markdown, info.CommitHashLong) || strings.Contains(markdown, "Branch:") || strings.Contains(markdown, "## Status") {
			t.Fatalf("commit Info template rendered incorrect fields:\n%s", markdown)
		}
	}
	for _, invalid := range []string{"HEAD~1", strings.Repeat("0", 40)} {
		if _, err := GitRefInfo(root, invalid, false); err == nil {
			t.Fatalf("GitRefInfo accepted invalid commit %q", invalid)
		}
	}
	markdown, err := RenderGitRefInfo(root, "v2.0.0", true)
	if err != nil {
		t.Fatalf("RenderGitRefInfo(tag) = %v", err)
	}
	if strings.Contains(markdown, "## Status") || strings.Contains(markdown, "## Worktree Path") || !strings.Contains(markdown, "Changes in commit:") {
		t.Fatalf("tag Info template rendered incorrect sections:\n%s", markdown)
	}
	if !strings.Contains(markdown, "- `v2.0.0`") || !strings.Contains(markdown, "- `v2.0.0-alias`") {
		t.Fatalf("tag Info template omitted tags at commit:\n%s", markdown)
	}

	rootInfo, err := GitRefInfo(root, rootBranch, false)
	if err != nil {
		t.Fatalf("GitRefInfo(root commit) = %v", err)
	}
	if len(rootInfo.RepoDiffFiles) != 1 || !strings.Contains(rootInfo.RepoDiffFiles[0].Diff, "+head contents") {
		t.Fatalf("root commit diff = %+v", rootInfo.RepoDiffFiles)
	}
	worktreeInfo, err := WorktreeInfo(root, root)
	if err != nil {
		t.Fatalf("WorktreeInfo() = %v", err)
	}
	if !worktreeInfo.IsWorktree || worktreeInfo.RepoStatus == "" {
		t.Fatalf("worktree Info fields = %+v", worktreeInfo)
	}
}

func TestRemoveWorktreeRejectsDirtyWorkingTreeWithoutForce(t *testing.T) {
	root := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	worktreePath := filepath.Join(t.TempDir(), "dirty-worktree")
	runGitTestCommand(t, gitPath, root, "worktree", "add", "--quiet", "-b", "dirty", worktreePath)
	if err := os.WriteFile(filepath.Join(worktreePath, "tracked file.txt"), []byte("modified\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RemoveWorktree(root, worktreePath); err == nil {
		t.Fatal("RemoveWorktree() error = nil, want dirty-worktree rejection")
	}
	if _, err := os.Stat(filepath.Join(worktreePath, "tracked file.txt")); err != nil {
		t.Fatalf("dirty worktree should remain after rejected removal: %v", err)
	}
}

func TestStageFileRejectsPathsOutsideWorkspace(t *testing.T) {
	root := initGitTestRepo(t)
	if err := StageFile(root, "../outside.txt"); err == nil {
		t.Fatal("StageFile() error = nil, want traversal rejection")
	}
	if _, err := ReadDiff(root, filepath.Join(root, "tracked file.txt")); err == nil {
		t.Fatal("ReadDiff() error = nil, want absolute path rejection")
	}
}

func TestReadDiffTreatsSymlinkAsReadOnly(t *testing.T) {
	root := initGitTestRepo(t)
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape.txt")); err != nil {
		t.Skipf("cannot create symlinks: %v", err)
	}
	diff, err := ReadDiff(root, "escape.txt")
	if err != nil {
		t.Fatalf("ReadDiff(symlink) error = %v", err)
	}
	if !diff.ReadOnly || diff.Working != outside {
		t.Fatalf("ReadDiff(symlink) = %+v, want read-only link target text", diff)
	}
}

func TestUnstageFileOnUnbornBranch(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	runGitTestCommand(t, gitPath, root, "init", "--quiet")
	if err := os.WriteFile(filepath.Join(root, "first.txt"), []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, gitPath, root, "add", "--", "first.txt")

	if err := UnstageFile(root, "first.txt"); err != nil {
		t.Fatalf("UnstageFile() on unborn branch error = %v", err)
	}
	status, err := WorkspaceStatus(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Files) != 1 || !status.Files[0].Untracked {
		t.Fatalf("status after unstage = %+v, want first.txt untracked", status.Files)
	}
}

func TestWorkspacePreservesWhitespaceInFilePaths(t *testing.T) {
	root := initGitTestRepo(t)
	const filePath = " leading name.txt "
	if err := os.WriteFile(filepath.Join(root, filePath), []byte("contents"), 0600); err != nil {
		t.Fatal(err)
	}
	status, err := WorkspaceStatus(root)
	if err != nil {
		t.Fatal(err)
	}
	if !filesForPath(status.Files, filePath).Untracked {
		t.Fatalf("status = %+v, want exact whitespace path %q", status.Files, filePath)
	}
	if err := StageFile(root, filePath); err != nil {
		t.Fatalf("StageFile(%q) error = %v", filePath, err)
	}
}

func TestReadDiffUsesOriginalPathForStagedRename(t *testing.T) {
	root := initGitTestRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	runGitTestCommand(t, gitPath, root, "mv", "tracked file.txt", "renamed file.txt")

	status, err := WorkspaceStatus(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Files) != 1 || status.Files[0].Path != "renamed file.txt" || status.Files[0].OriginalPath != "tracked file.txt" {
		t.Fatalf("rename status = %+v", status.Files)
	}
	diff, err := ReadDiff(root, status.Files[0].Path, status.Files[0].OriginalPath)
	if err != nil {
		t.Fatalf("ReadDiff(rename) error = %v", err)
	}
	if diff.Original != "head contents\n" || diff.Working != "head contents\n" {
		t.Fatalf("ReadDiff(rename) = %+v, want content from original HEAD path", diff)
	}
}

func filesForPath(files []FileStatus, path string) FileStatus {
	for _, file := range files {
		if file.Path == path {
			return file
		}
	}
	return FileStatus{}
}

func resolveTestPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(resolved)
}
