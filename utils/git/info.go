package git

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/lmorg/ttyphoon/utils/markdownutil"
)

//go:embed info.md
var infoTemplate string

type InfoDiffFile struct {
	Filename string
	Diff     string
}

type Info struct {
	CommitHashLong   string
	BranchName       string
	MatchedTags      []string
	IsWorktree       bool
	RepoWorktreePath string
	RepoStatus       string
	RepoDiffFiles    []InfoDiffFile
	GitExeVersion    string
	GitExePath       string
}

func WorktreeInfo(root, worktreePath string) (Info, error) {
	_, repoRoot, err := checkedRepository(root)
	if err != nil {
		return Info{}, err
	}
	target, err := resolveExistingPathPrefix(worktreePath)
	if err != nil {
		return Info{}, err
	}
	worktrees, err := ListWorktrees(repoRoot)
	if err != nil {
		return Info{}, err
	}
	registered := false
	for _, worktree := range worktrees {
		listedPath, pathErr := resolveExistingPathPrefix(worktree.Path)
		if pathErr == nil && samePath(listedPath, target) {
			registered = true
			break
		}
	}
	if !registered {
		return Info{}, errors.New("selected path is not a registered worktree for the current repository")
	}

	gitPath, worktreeRoot, err := checkedRepository(target)
	if err != nil {
		return Info{}, err
	}
	info := Info{RepoWorktreePath: worktreeRoot, GitExePath: gitPath, IsWorktree: true}
	if err := populateCommitInfo(gitPath, worktreeRoot, "HEAD", &info); err != nil {
		return Info{}, err
	}
	if branch, branchErr := runGit(gitPath, worktreeRoot, "symbolic-ref", "--quiet", "--short", "HEAD"); branchErr == nil {
		info.BranchName = strings.TrimSpace(string(branch))
	}

	status, err := runGit(gitPath, worktreeRoot, "status")
	if err != nil {
		return Info{}, err
	}
	info.RepoStatus = strings.TrimSpace(string(status))

	// Without a HEAD commit there is nothing to compare against, so show staged changes.
	base := []string{"HEAD"}
	if _, err := runGit(gitPath, worktreeRoot, "rev-parse", "--verify", "HEAD"); err != nil {
		base = []string{"--cached"}
	}
	names, err := runGit(gitPath, worktreeRoot, append(append([]string{"diff", "--name-only", "-z"}, base...), "--")...)
	if err != nil {
		return Info{}, err
	}
	for _, name := range bytes.Split(names, []byte{0}) {
		if len(name) == 0 {
			continue
		}
		args := append(append([]string{"diff"}, base...), "--", string(name))
		diff, err := runGit(gitPath, worktreeRoot, args...)
		if err != nil {
			return Info{}, err
		}
		if len(diff) > maxDiffBytes {
			diff = append(diff[:maxDiffBytes], []byte("\n… diff truncated")...)
		}
		info.RepoDiffFiles = append(info.RepoDiffFiles, InfoDiffFile{
			Filename: string(name),
			Diff:     strings.TrimRight(string(diff), "\n"),
		})
	}
	return info, nil
}

func GitRefInfo(root, name string, isTag bool) (Info, error) {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return Info{}, err
	}
	var ref string
	if isTag {
		ref, err = checkedLocalTag(gitPath, repoRoot, name)
	} else {
		ref, err = checkedComparisonRef(gitPath, repoRoot, name)
	}
	if err != nil {
		return Info{}, err
	}
	commit, err := runGit(gitPath, repoRoot, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	if err != nil {
		return Info{}, fmt.Errorf("Git ref %q does not resolve to a commit", name)
	}
	info := Info{GitExePath: gitPath}
	if !isTag && strings.HasPrefix(ref, "refs/heads/") {
		info.BranchName = strings.TrimPrefix(ref, "refs/heads/")
	}
	if err := populateCommitInfo(gitPath, repoRoot, strings.TrimSpace(string(commit)), &info); err != nil {
		return Info{}, err
	}
	if err := populateCommitDiff(gitPath, repoRoot, info.CommitHashLong, &info); err != nil {
		return Info{}, err
	}
	return info, nil
}

func populateCommitInfo(gitPath, repoRoot, revision string, info *Info) error {
	version, err := runGit(gitPath, repoRoot, "--version")
	if err != nil {
		return err
	}
	info.GitExeVersion = strings.TrimSpace(string(version))
	commit, err := runGit(gitPath, repoRoot, "rev-parse", "--verify", revision+"^{commit}")
	if err != nil {
		return err
	}
	info.CommitHashLong = strings.TrimSpace(string(commit))
	tags, err := runGit(gitPath, repoRoot, "tag", "--points-at", info.CommitHashLong)
	if err != nil {
		return err
	}
	info.MatchedTags = strings.Fields(string(tags))
	return nil
}

func populateCommitDiff(gitPath, repoRoot, commit string, info *Info) error {
	parentOutput, err := runGit(gitPath, repoRoot, "rev-list", "--parents", "-n", "1", commit)
	if err != nil {
		return err
	}
	parentFields := strings.Fields(string(parentOutput))
	parent := ""
	if len(parentFields) > 1 {
		parent = parentFields[1]
	}
	var names []byte
	if parent == "" {
		names, err = runGit(gitPath, repoRoot, "diff-tree", "--root", "--no-commit-id", "--name-only", "-z", "--no-renames", "-r", commit, "--")
	} else {
		names, err = runGit(gitPath, repoRoot, "diff", "--name-only", "-z", "--no-renames", parent, commit, "--")
	}
	if err != nil {
		return err
	}
	for _, name := range bytes.Split(names, []byte{0}) {
		if len(name) == 0 {
			continue
		}
		var diff []byte
		if parent == "" {
			diff, err = runGit(gitPath, repoRoot, "show", "--format=", "--no-ext-diff", "--root", commit, "--", string(name))
		} else {
			diff, err = runGit(gitPath, repoRoot, "diff", "--no-ext-diff", parent, commit, "--", string(name))
		}
		if err != nil {
			return err
		}
		if len(diff) > maxDiffBytes {
			diff = append(diff[:maxDiffBytes], []byte("\n… diff truncated")...)
		}
		info.RepoDiffFiles = append(info.RepoDiffFiles, InfoDiffFile{
			Filename: string(name),
			Diff:     strings.TrimRight(string(diff), "\n"),
		})
	}
	return nil
}

func RenderWorktreeInfo(root, worktreePath string) (string, error) {
	info, err := WorktreeInfo(root, worktreePath)
	if err != nil {
		return "", err
	}
	return renderInfo(info)
}

func RenderGitRefInfo(root, name string, isTag bool) (string, error) {
	info, err := GitRefInfo(root, name, isTag)
	if err != nil {
		return "", err
	}
	return renderInfo(info)
}

func renderInfo(info Info) (string, error) {
	tmpl, err := template.New("info").Funcs(markdownutil.TemplateFuncMap()).Parse(infoTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, info); err != nil {
		return "", err
	}
	return buf.String(), nil
}
