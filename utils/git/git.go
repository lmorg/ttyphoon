package git

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/utils/file"
	"golang.org/x/mod/semver"
)

const maxDiffBytes = 4 << 20

var aheadBehindPattern = regexp.MustCompile(`\[ahead ([0-9]+)(?:, behind ([0-9]+))?\]|\[behind ([0-9]+)(?:, ahead ([0-9]+))?\]`)
var commitHashPattern = regexp.MustCompile(`^[0-9a-fA-F]{7,64}$`)

type FileStatus struct {
	Path         string `json:"path"`
	OriginalPath string `json:"originalPath,omitempty"`
	Code         string `json:"code"`
	CompareOnly  bool   `json:"compareOnly,omitempty"`
	Staged       bool   `json:"staged"`
	Unstaged     bool   `json:"unstaged"`
	Untracked    bool   `json:"untracked"`
	Conflict     bool   `json:"conflict"`
}

type Status struct {
	Root   string       `json:"root"`
	Branch string       `json:"branch"`
	Ahead  int          `json:"ahead"`
	Behind int          `json:"behind"`
	Clean  bool         `json:"clean"`
	Files  []FileStatus `json:"files"`
}

type Diff struct {
	Path     string `json:"path"`
	Original string `json:"original"`
	Working  string `json:"working"`
	Binary   bool   `json:"binary"`
	Deleted  bool   `json:"deleted"`
	ReadOnly bool   `json:"readOnly"`
}

type Worktree struct {
	Path   string `json:"path"`
	Branch string `json:"branch"`
}

type BrowserLink struct {
	Provider string `json:"provider"`
	URL      string `json:"url"`
}

type Commit struct {
	Hash     string   `json:"hash"`
	FullHash string   `json:"fullHash"`
	Subject  string   `json:"subject"`
	Message  string   `json:"message"`
	Date     string   `json:"date"`
	Author   string   `json:"author"`
	Tags     []string `json:"tags"`
	Files    []string `json:"files,omitempty"`
	Graph    []string `json:"graph"`
}

func ListCommits(root string, limit int) ([]Commit, error) {
	return listCommits(root, limit, false)
}

func ListCommitsWithFiles(root string, limit int) ([]Commit, error) {
	return listCommits(root, limit, true)
}

func listCommits(root string, limit int, includeFiles bool) ([]Commit, error) {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return nil, err
	}

	if limit < 1 || limit > 10000 {
		return nil, errors.New("commit history limit must be between 1 and 10000")
	}
	head, err := runGit(gitPath, repoRoot, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return []Commit{}, nil
	}
	headRef := strings.TrimSpace(string(head))
	args := []string{"log", "--graph", "--topo-order", "--no-color", "--decorate=short", "--decorate-refs=refs/tags/*", "--max-count=" + strconv.Itoa(limit), "--date=format:%Y-%m-%d %H:%M:%S", "--format=%x00%h%x00%s%x00%ad%x00%an%x00%H%x00%D%x00", headRef, "--"}
	output, err := runGit(gitPath, repoRoot, args...)
	if err != nil {
		return nil, err
	}
	commits := make([]Commit, 0)
	for _, line := range strings.Split(strings.TrimRight(string(output), "\n"), "\n") {
		fields := strings.Split(line, "\x00")
		if len(fields) == 8 {
			tags := make([]string, 0)
			for _, decoration := range strings.Split(fields[6], ", ") {
				if tag, ok := strings.CutPrefix(decoration, "tag: "); ok {
					tags = append(tags, tag)
				}
			}
			sort.Strings(tags)
			commits = append(commits, Commit{Hash: fields[1], FullHash: fields[5], Subject: fields[2], Date: fields[3], Author: fields[4], Tags: tags, Graph: []string{fields[0]}})
		} else if len(commits) > 0 && strings.TrimSpace(line) != "" {
			last := &commits[len(commits)-1]
			last.Graph = append(last.Graph, line)
		}
	}
	messageOutput, err := runGit(gitPath, repoRoot, "log", "--topo-order", "--no-color", "--max-count="+strconv.Itoa(limit), "--format=%H%x00%B%x00", headRef, "--")
	if err != nil {
		return nil, err
	}
	messageFields := strings.Split(string(messageOutput), "\x00")
	messages := make(map[string]string, len(commits))
	for index := 0; index+1 < len(messageFields); index += 2 {
		messages[strings.TrimSpace(messageFields[index])] = strings.TrimRight(messageFields[index+1], "\r\n")
	}
	for index := range commits {
		commits[index].Message = messages[commits[index].FullHash]
	}
	if includeFiles {
		output, err := runGit(gitPath, repoRoot, "log", "--topo-order", "--no-color", "--no-ext-diff", "--raw", "--root", "-m", "--no-renames", "-z", "--format=%H%x00", "--max-count="+strconv.Itoa(limit), headRef, "--")
		if err != nil {
			return nil, err
		}
		byHash := make(map[string]*Commit, len(commits))
		seen := make(map[string]map[string]bool, len(commits))
		for index := range commits {
			byHash[commits[index].FullHash] = &commits[index]
			seen[commits[index].FullHash] = make(map[string]bool)
		}
		var current *Commit
		records := strings.Split(string(output), "\x00")
		for index := 0; index < len(records); index++ {
			record := strings.TrimLeft(records[index], "\r\n")
			if commit := byHash[strings.TrimSpace(record)]; commit != nil {
				current = commit
			} else if strings.HasPrefix(record, ":") && index+1 < len(records) {
				index++
				path := records[index]
				if current != nil && path != "" && !seen[current.FullHash][path] {
					current.Files = append(current.Files, path)
					seen[current.FullHash][path] = true
				}
			}
		}
	}
	return commits, nil
}

func ListWorktrees(root string) ([]Worktree, error) {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return nil, err
	}
	output, err := runGit(gitPath, repoRoot, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return nil, err
	}

	worktrees := make([]Worktree, 0)
	var current Worktree
	var head string
	flush := func() {
		if current.Path != "" {
			worktrees = append(worktrees, current)
			current = Worktree{}
			head = ""
		}
	}
	for _, record := range bytes.Split(output, []byte{0}) {
		line := string(record)
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "worktree "):
			flush()
			current.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch "):
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		case strings.HasPrefix(line, "HEAD "):
			head = strings.TrimPrefix(line, "HEAD ")
		case line == "detached":
			if tags, tagsErr := runGit(gitPath, repoRoot, "tag", "--points-at", head); tagsErr == nil {
				pointedTags := strings.Fields(string(tags))
				sort.Slice(pointedTags, func(i, j int) bool { return len(pointedTags[i]) > len(pointedTags[j]) })
				for _, tag := range pointedTags {
					if strings.HasSuffix(filepath.ToSlash(current.Path), "/"+tag) {
						current.Branch = "tag: " + tag
						break
					}
				}
			}
			if current.Branch == "" {
				if tag, tagErr := runGit(gitPath, repoRoot, "describe", "--tags", "--exact-match", head); tagErr == nil {
					current.Branch = "tag: " + strings.TrimSpace(string(tag))
				} else {
					shortHash, err := runGit(gitPath, repoRoot, "rev-parse", "--short", head)
					if err != nil {
						return nil, err
					}
					current.Branch = strings.TrimSpace(string(shortHash))
				}
			}
		}
	}
	flush()
	sort.Slice(worktrees, func(i, j int) bool {
		return strings.ToLower(worktrees[i].Path) < strings.ToLower(worktrees[j].Path)
	})
	return worktrees, nil
}

func ListBranches(root string) ([]string, error) {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return nil, err
	}
	output, err := runGit(gitPath, repoRoot, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return nil, err
	}
	branches := strings.Fields(string(output))
	sort.Strings(branches)
	return branches, nil
}

func ListTags(root string) ([]string, error) {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return nil, err
	}
	output, err := runGit(gitPath, repoRoot, "for-each-ref", "--format=%(refname:short)", "refs/tags")
	if err != nil {
		return nil, err
	}
	tags := strings.Fields(string(output))
	sort.Slice(tags, func(i, j int) bool {
		leftValid, rightValid := semver.IsValid(tags[i]), semver.IsValid(tags[j])
		if leftValid != rightValid {
			return leftValid
		}
		if leftValid {
			if comparison := semver.Compare(tags[i], tags[j]); comparison != 0 {
				return comparison > 0
			}
		}
		return tags[i] > tags[j]
	})
	return tags, nil
}

func CreateTag(root, tag string) error {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return err
	}
	if _, err := checkedTagName(gitPath, repoRoot, tag); err != nil {
		return err
	}
	_, err = runGit(gitPath, repoRoot, "tag", "--", tag)
	return err
}

func DeleteTag(root, tag string) error {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return err
	}
	if _, err := checkedLocalTag(gitPath, repoRoot, tag); err != nil {
		return err
	}
	_, err = runGit(gitPath, repoRoot, "tag", "--delete", "--", tag)
	return err
}

func CheckoutTag(root, tag string) error {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return err
	}
	ref, err := checkedLocalTag(gitPath, repoRoot, tag)
	if err != nil {
		return err
	}
	_, err = runGit(gitPath, repoRoot, "checkout", "--detach", ref)
	return err
}

func CreateTagWorktree(root, tag, worktreePath string) (Worktree, error) {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return Worktree{}, err
	}
	ref, err := checkedLocalTag(gitPath, repoRoot, tag)
	if err != nil {
		return Worktree{}, err
	}
	commit, err := runGit(gitPath, repoRoot, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	if err != nil {
		return Worktree{}, fmt.Errorf("tag %q does not resolve to a commit", tag)
	}
	worktree, err := CreateWorktree(repoRoot, strings.TrimSpace(string(commit)), worktreePath, false)
	if err != nil {
		return Worktree{}, err
	}
	worktree.Branch = "tag: " + tag
	return worktree, nil
}

func PullTags(root string) (map[string]string, error) {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return nil, err
	}
	return runGitWithOutput(gitPath, repoRoot, "fetch", "origin", "refs/tags/*:refs/tags/*")
}

func PushTag(root, tag string) (map[string]string, error) {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return nil, err
	}
	ref, err := checkedLocalTag(gitPath, repoRoot, tag)
	if err != nil {
		return nil, err
	}
	return runGitWithOutput(gitPath, repoRoot, "push", "origin", ref)
}

func checkedTagName(gitPath, repoRoot, tag string) (string, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return "", errors.New("tag name is empty")
	}
	ref := "refs/tags/" + tag
	if _, err := runGit(gitPath, repoRoot, "check-ref-format", ref); err != nil {
		return "", fmt.Errorf("invalid tag name %q", tag)
	}
	return ref, nil
}

func checkedLocalTag(gitPath, repoRoot, tag string) (string, error) {
	ref, err := checkedTagName(gitPath, repoRoot, tag)
	if err != nil {
		return "", err
	}
	if _, err := runGit(gitPath, repoRoot, "rev-parse", "--verify", "--quiet", ref); err != nil {
		return "", fmt.Errorf("local tag %q does not exist", tag)
	}
	return ref, nil
}

func CheckoutBranch(root, branch string, create bool) error {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return err
	}
	if _, err := runGit(gitPath, repoRoot, "check-ref-format", "--branch", branch); err != nil {
		return fmt.Errorf("invalid branch name %q", branch)
	}
	if !create {
		if _, err := checkedLocalBranch(gitPath, repoRoot, branch); err != nil {
			return err
		}
	}
	args := []string{"checkout", "--no-guess"}
	if create {
		args = append(args, "-b")
	}
	args = append(args, branch, "--")
	_, err = runGit(gitPath, repoRoot, args...)
	return err
}

func DeleteBranch(root, branch string) error {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return err
	}
	ref, err := checkedLocalBranch(gitPath, repoRoot, branch)
	if err != nil {
		return err
	}
	_, err = runGit(gitPath, repoRoot, "branch", "-d", "--", strings.TrimPrefix(ref, "refs/heads/"))
	return err
}

func FetchBranch(root, branch string) (map[string]string, error) {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return nil, err
	}
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
	return runGitWithOutput(gitPath, repoRoot, "fetch", "--", remote, remoteRef+":"+destination)
}

func runGitWithOutput(gitPath, repoRoot string, args ...string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, gitPath, append([]string{"-C", repoRoot}, args...)...)
	cmd.Env = gitCommandEnvironment()
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	result := map[string]string{"stdout": strings.TrimSpace(stdout.String()), "stderr": strings.TrimSpace(stderr.String()), "error": ""}
	if ctx.Err() != nil {
		result["error"] = "git command timed out"
	} else if runErr != nil {
		result["error"] = runErr.Error()
	}
	if result["error"] == "" {
		for _, output := range []string{result["stdout"], result["stderr"]} {
			for _, line := range strings.Split(output, "\n") {
				if line != "" {
					log.Printf("[info] %s", line)
				}
			}
		}
	}
	return result, nil
}

func gitCommandEnvironment() []string {
	environment := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if name != "GIT_TERMINAL_PROMPT" {
			environment = append(environment, entry)
		}
	}
	return append(environment, "GIT_TERMINAL_PROMPT=0")
}

func DefaultWorktreePath(root string) (string, error) {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return "", err
	}
	hashSource := repoRoot
	if output, err := runGit(gitPath, repoRoot, "config", "--get", "remote.origin.url"); err == nil {
		if origin := strings.TrimSpace(string(output)); origin != "" {
			hashSource = origin
		}
	}
	hash := sha256.Sum256([]byte(hashSource))
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(file.DocumentsDir(home), ".git_worktrees", hex.EncodeToString(hash[:])), nil
}

func CreateWorktree(root, branch, worktreePath string, createBranch bool) (Worktree, error) {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return Worktree{}, err
	}
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return Worktree{}, errors.New("branch name is empty")
	}
	detached := !createBranch && commitHashPattern.MatchString(branch)
	var detachedLabel string
	if detached {
		if _, err := runGit(gitPath, repoRoot, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch+"^{commit}"); err == nil {
			detached = false
		}
	}
	if detached {
		output, err := runGit(gitPath, repoRoot, "rev-parse", "--verify", "--quiet", branch+"^{commit}")
		if err != nil {
			return Worktree{}, fmt.Errorf("worktree commit %q does not exist or is ambiguous", branch)
		}
		branch = strings.TrimSpace(string(output))
		shortHash, err := runGit(gitPath, repoRoot, "rev-parse", "--short", branch)
		if err != nil {
			return Worktree{}, err
		}
		detachedLabel = strings.TrimSpace(string(shortHash))
	} else if _, err := runGit(gitPath, repoRoot, "check-ref-format", "refs/heads/"+branch); err != nil {
		return Worktree{}, fmt.Errorf("invalid branch name %q", branch)
	}
	target := worktreePath
	if !filepath.IsAbs(target) {
		relPath, pathErr := validateRelativePath(target)
		if pathErr != nil {
			return Worktree{}, pathErr
		}
		target = filepath.Join(repoRoot, relPath)
	}
	target = filepath.Clean(target)
	if filepath.IsAbs(target) {
		target, err = resolveExistingPathPrefix(target)
		if err != nil {
			return Worktree{}, err
		}
	}
	relPath, relErr := filepath.Rel(repoRoot, target)
	if relErr != nil {
		return Worktree{}, relErr
	}
	insideRepository := relPath != ".." && !strings.HasPrefix(relPath, ".."+string(filepath.Separator)) && !filepath.IsAbs(relPath)
	if relPath == "." {
		return Worktree{}, errors.New("worktree path cannot be the current repository root")
	}
	if insideRepository {
		relPath, err = validateRelativePath(relPath)
		if err != nil {
			return Worktree{}, err
		}
		if strings.EqualFold(relPath, ".git") || strings.HasPrefix(strings.ToLower(relPath), ".git"+string(filepath.Separator)) {
			return Worktree{}, errors.New("worktree path cannot be inside .git")
		}
	}
	if err := ensureWorktreeParent(filepath.Dir(target)); err != nil {
		return Worktree{}, err
	}
	if _, err := os.Lstat(target); err == nil {
		return Worktree{}, fmt.Errorf("worktree path %q already exists", relPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Worktree{}, err
	}
	if insideRepository {
		if err := excludeWorktreePath(gitPath, repoRoot, relPath); err != nil {
			return Worktree{}, err
		}
	}

	args := []string{"worktree", "add"}
	if detached {
		args = append(args, "--detach", target, branch)
	} else if createBranch {
		args = append(args, "-b", branch, target)
	} else {
		args = append(args, target, branch)
	}
	if _, err := runGit(gitPath, repoRoot, args...); err != nil {
		return Worktree{}, err
	}
	if detached {
		return Worktree{Path: target, Branch: detachedLabel}, nil
	}
	return Worktree{Path: target, Branch: branch}, nil
}

func RemoveWorktree(root, worktreePath string) error {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return err
	}
	target, err := resolveExistingPathPrefix(worktreePath)
	if err != nil {
		return err
	}
	worktrees, err := ListWorktrees(repoRoot)
	if err != nil {
		return err
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
		return errors.New("selected path is not a registered worktree for the current repository")
	}
	if _, err := runGit(gitPath, repoRoot, "worktree", "remove", "--", target); err != nil {
		return err
	}
	return nil
}

func excludeWorktreePath(gitPath, repoRoot, relativePath string) error {
	output, err := runGit(gitPath, repoRoot, "rev-parse", "--git-common-dir")
	if err != nil {
		return err
	}
	commonDir := strings.TrimRight(string(output), "\r\n")
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(repoRoot, commonDir)
	}
	excludePath := filepath.Join(commonDir, "info", "exclude")
	existing, err := os.ReadFile(excludePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	rule := "/" + escapeGitExcludePattern(filepath.ToSlash(relativePath)) + "/"
	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSuffix(line, "\r") == rule {
			return nil
		}
	}
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		existing = append(existing, '\n')
	}
	existing = append(existing, []byte(rule+"\n")...)
	return os.WriteFile(excludePath, existing, 0600)
}

func escapeGitExcludePattern(path string) string {
	return strings.NewReplacer("\\", "\\\\", "*", "\\*", "?", "\\?", "[", "\\[").Replace(path)
}

func ensureWorktreeParent(parent string) error {
	parent = filepath.Clean(parent)
	volume := filepath.VolumeName(parent)
	current := volume + string(filepath.Separator)
	if volume == "" {
		current = string(filepath.Separator)
	}
	relativeParent := strings.TrimPrefix(parent, current)
	if relativeParent == "" || relativeParent == "." {
		return nil
	}
	for _, segment := range strings.Split(relativeParent, string(filepath.Separator)) {
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(current, 0700); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("worktree parent %q must be a real directory", current)
		}
	}
	return nil
}

func resolveExistingPathPrefix(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	path = filepath.Clean(path)
	var missing []string
	for {
		if _, err := os.Lstat(path); err == nil {
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", fmt.Errorf("cannot resolve worktree destination %q", path)
		}
		missing = append(missing, filepath.Base(path))
		path = parent
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	for index := len(missing) - 1; index >= 0; index-- {
		path = filepath.Join(path, missing[index])
	}
	return filepath.Clean(path), nil
}

func WorkspaceStatus(root string) (Status, error) {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return Status{}, err
	}

	output, err := runGit(gitPath, repoRoot, "--no-optional-locks", "status", "--porcelain=v1", "-z", "--branch", "--untracked-files=all")
	if err != nil {
		return Status{}, err
	}

	status, err := parseStatus(output)
	if err != nil {
		return Status{}, err
	}
	status.Root = repoRoot
	status.Clean = len(status.Files) == 0
	return status, nil
}

func BranchDiffFiles(root, branch string) ([]FileStatus, error) {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return nil, err
	}
	branchRef, err := checkedComparisonRef(gitPath, repoRoot, branch)
	if err != nil {
		return nil, err
	}
	output, err := runGit(gitPath, repoRoot, "diff", "--name-only", "-z", "--no-renames", branchRef, "--")
	if err != nil {
		return nil, err
	}
	status, err := WorkspaceStatus(repoRoot)
	if err != nil {
		return nil, err
	}
	workingFiles := make(map[string]FileStatus, len(status.Files))
	for _, file := range status.Files {
		workingFiles[file.Path] = file
	}

	files := make([]FileStatus, 0)
	seen := make(map[string]struct{})
	for _, rawPath := range bytes.Split(output, []byte{0}) {
		if len(rawPath) == 0 {
			continue
		}
		path := filepath.ToSlash(string(rawPath))
		if _, ok := seen[path]; ok {
			continue
		}
		file, ok := workingFiles[path]
		if !ok {
			file = FileStatus{Path: path, Code: " M"}
		}
		file.CompareOnly = true
		files = append(files, file)
		seen[path] = struct{}{}
	}
	for _, file := range status.Files {
		if !file.Untracked {
			continue
		}
		if _, ok := seen[file.Path]; ok {
			continue
		}
		file.CompareOnly = true
		files = append(files, file)
		seen[file.Path] = struct{}{}
	}
	tree, err := runGit(gitPath, repoRoot, "ls-tree", "-r", "--name-only", "-z", branchRef, "--")
	if err != nil {
		return nil, err
	}
	for _, rawPath := range bytes.Split(tree, []byte{0}) {
		if len(rawPath) == 0 {
			continue
		}
		path := filepath.ToSlash(string(rawPath))
		if _, ok := seen[path]; ok {
			continue
		}
		file := workingFiles[path]
		file.Path = path
		file.Code = "="
		file.CompareOnly = true
		files = append(files, file)
		seen[path] = struct{}{}
	}
	sort.Slice(files, func(i, j int) bool {
		if (files[i].Code == "=") != (files[j].Code == "=") {
			return files[j].Code == "="
		}
		return files[i].Path < files[j].Path
	})
	return files, nil
}

func checkedLocalBranch(gitPath, repoRoot, branch string) (string, error) {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return "", errors.New("comparison branch is empty")
	}
	ref := "refs/heads/" + branch
	if _, err := runGit(gitPath, repoRoot, "check-ref-format", ref); err != nil {
		return "", fmt.Errorf("invalid comparison branch %q", branch)
	}
	if _, err := runGit(gitPath, repoRoot, "rev-parse", "--verify", "--quiet", ref+"^{commit}"); err != nil {
		return "", fmt.Errorf("comparison branch %q does not exist", branch)
	}
	return ref, nil
}

func checkedComparisonRef(gitPath, repoRoot, target string) (string, error) {
	target = strings.TrimSpace(target)
	ref, branchErr := checkedLocalBranch(gitPath, repoRoot, target)
	if branchErr == nil {
		return ref, nil
	}
	if !commitHashPattern.MatchString(target) {
		return "", branchErr
	}
	output, err := runGit(gitPath, repoRoot, "rev-parse", "--verify", "--quiet", target+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("comparison commit %q does not exist or is ambiguous", target)
	}
	return strings.TrimSpace(string(output)), nil
}

func MetadataDirectories(root string) ([]string, error) {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return nil, err
	}
	output, err := runGit(gitPath, repoRoot, "rev-parse", "--git-dir", "--git-common-dir")
	if err != nil {
		return nil, err
	}

	directories := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)
	for _, directory := range strings.Split(strings.TrimRight(string(output), "\r\n"), "\n") {
		directory = strings.TrimSuffix(directory, "\r")
		if directory == "" {
			continue
		}
		if !filepath.IsAbs(directory) {
			directory = filepath.Join(repoRoot, directory)
		}
		directory, err = filepath.Abs(directory)
		if err != nil {
			return nil, err
		}
		directory = filepath.Clean(directory)
		if resolved, resolveErr := filepath.EvalSymlinks(directory); resolveErr == nil {
			directory = filepath.Clean(resolved)
		}
		if _, ok := seen[directory]; ok {
			continue
		}
		seen[directory] = struct{}{}
		directories = append(directories, directory)
	}
	return directories, nil
}

func RemoteBrowserLink(root, branch string) (BrowserLink, error) {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return BrowserLink{}, err
	}
	if strings.TrimSpace(branch) == "" || branch == "HEAD" {
		return BrowserLink{}, nil
	}
	output, err := runGit(gitPath, repoRoot, "remote", "get-url", "origin")
	if err != nil {
		return BrowserLink{}, nil
	}
	return browserLinkForRemote(strings.TrimSpace(string(output)), branch), nil
}

func browserLinkForRemote(remote, branch string) BrowserLink {
	if strings.TrimSpace(branch) == "" || branch == "HEAD" {
		return BrowserLink{}
	}
	host, remotePath := parseRemoteAddress(remote)
	if host == "" {
		return BrowserLink{}
	}
	parts := strings.Split(strings.Trim(remotePath, "/"), "/")
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return BrowserLink{}
	}
	parts[len(parts)-1] = strings.TrimSuffix(parts[len(parts)-1], ".git")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return BrowserLink{}
		}
	}

	branchPath := escapeURLPath(strings.Split(branch, "/"))
	path := escapeURLPath(parts)
	switch host {
	case "github.com":
		if len(parts) == 2 {
			return BrowserLink{Provider: "github", URL: "https://github.com/" + path + "/tree/" + branchPath}
		}
	case "gitlab.com":
		if len(parts) >= 2 {
			return BrowserLink{Provider: "gitlab", URL: "https://gitlab.com/" + path + "/-/tree/" + branchPath}
		}
	case "bitbucket.org":
		if len(parts) == 2 {
			return BrowserLink{Provider: "bitbucket", URL: "https://bitbucket.org/" + path + "/src/" + branchPath + "/"}
		}
	case "dev.azure.com":
		if len(parts) == 4 && parts[2] == "_git" {
			return azureBrowserLink(parts[0], parts[1], parts[3], branch)
		}
	case "ssh.dev.azure.com", "vs-ssh.visualstudio.com":
		if len(parts) == 4 && parts[0] == "v3" {
			return azureBrowserLink(parts[1], parts[2], parts[3], branch)
		}
	default:
		if strings.HasSuffix(host, ".visualstudio.com") {
			organization := strings.TrimSuffix(host, ".visualstudio.com")
			if len(parts) == 3 && parts[1] == "_git" {
				return azureBrowserLink(organization, parts[0], parts[2], branch)
			}
			if len(parts) == 4 && parts[0] == "v3" {
				return azureBrowserLink(parts[1], parts[2], parts[3], branch)
			}
		}
	}
	return BrowserLink{}
}

func parseRemoteAddress(remote string) (host, remotePath string) {
	if strings.Contains(remote, "://") {
		parsed, err := url.Parse(remote)
		if err != nil {
			return "", ""
		}
		return strings.ToLower(parsed.Hostname()), parsed.Path
	}

	address := remote
	if at := strings.LastIndex(address, "@"); at >= 0 {
		address = address[at+1:]
	}
	colon := strings.Index(address, ":")
	if colon < 0 || strings.Contains(address[:colon], "/") {
		return "", ""
	}
	return strings.ToLower(address[:colon]), address[colon+1:]
}

func azureBrowserLink(organization, project, repository, branch string) BrowserLink {
	path := escapeURLPath([]string{organization, project, "_git", repository})
	query := url.Values{"version": {"GB" + branch}}.Encode()
	return BrowserLink{Provider: "azuredevops", URL: "https://dev.azure.com/" + path + "?" + query}
}

func escapeURLPath(parts []string) string {
	escaped := make([]string, len(parts))
	for index, part := range parts {
		escaped[index] = url.PathEscape(part)
	}
	return strings.Join(escaped, "/")
}

func ReadDiff(root, filePath string, originalPaths ...string) (Diff, error) {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return Diff{}, err
	}
	relPath, err := validateRelativePath(filePath)
	if err != nil {
		return Diff{}, err
	}

	result := Diff{Path: filepath.ToSlash(relPath)}
	headPath := relPath
	if len(originalPaths) > 0 && strings.TrimSpace(originalPaths[0]) != "" {
		headPath, err = validateRelativePath(originalPaths[0])
		if err != nil {
			return Diff{}, err
		}
	}
	baseRef := "HEAD"
	if len(originalPaths) > 1 && strings.TrimSpace(originalPaths[1]) != "" {
		baseRef, err = checkedComparisonRef(gitPath, repoRoot, originalPaths[1])
		if err != nil {
			return Diff{}, err
		}
	}
	heading := baseRef + ":" + filepath.ToSlash(headPath)
	if _, err := runGit(gitPath, repoRoot, "rev-parse", "--verify", baseRef); err == nil {
		original, showErr := runGit(gitPath, repoRoot, "show", heading)
		if showErr == nil {
			if len(original) > maxDiffBytes {
				return Diff{}, fmt.Errorf("file exceeds the %d MiB diff limit", maxDiffBytes>>20)
			}
			result.Original = string(original)
		} else if !isMissingGitObject(showErr) {
			return Diff{}, showErr
		}
	}

	workingPath := filepath.Join(repoRoot, relPath)
	fileInfo, statErr := os.Lstat(workingPath)
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return Diff{}, statErr
	}
	result.Deleted = errors.Is(statErr, os.ErrNotExist)
	var working []byte
	if !result.Deleted {
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			linkTarget, linkErr := os.Readlink(workingPath)
			if linkErr != nil {
				return Diff{}, linkErr
			}
			working = []byte(linkTarget)
			result.ReadOnly = true
		} else if !fileInfo.Mode().IsRegular() {
			return Diff{}, fmt.Errorf("cannot diff non-regular file %q", relPath)
		} else {
			working, err = os.ReadFile(workingPath)
			if err != nil {
				return Diff{}, err
			}
		}
		if len(working) > maxDiffBytes {
			return Diff{}, fmt.Errorf("file exceeds the %d MiB diff limit", maxDiffBytes>>20)
		}
		result.Working = string(working)
	}

	result.Binary = bytes.IndexByte([]byte(result.Original), 0) >= 0 || bytes.IndexByte(working, 0) >= 0
	if result.Binary {
		result.Original = ""
		result.Working = ""
		result.ReadOnly = true
	}
	return result, nil
}

func StageFile(root, filePath string) error {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return err
	}
	relPath, err := validateRelativePath(filePath)
	if err != nil {
		return err
	}
	_, err = runGit(gitPath, repoRoot, "add", "--", filepath.ToSlash(relPath))
	return err
}

func StageAllTrackedFiles(root string) error {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return err
	}
	_, err = runGit(gitPath, repoRoot, "add", "-u", "--", ".")
	return err
}

func UnstageFile(root, filePath string) error {
	gitPath, repoRoot, err := checkedRepository(root)
	if err != nil {
		return err
	}
	relPath, err := validateRelativePath(filePath)
	if err != nil {
		return err
	}
	_, restoreErr := runGit(gitPath, repoRoot, "restore", "--staged", "--", filepath.ToSlash(relPath))
	if restoreErr == nil {
		return nil
	}
	if _, headErr := runGit(gitPath, repoRoot, "rev-parse", "--verify", "HEAD"); headErr != nil {
		_, rmErr := runGit(gitPath, repoRoot, "rm", "--cached", "--", filepath.ToSlash(relPath))
		if rmErr == nil {
			return nil
		}
	}
	return restoreErr
}

func checkedRepository(root string) (gitPath, repoRoot string, err error) {
	gitPath, err = exec.LookPath("git")
	if err != nil {
		return "", "", fmt.Errorf("git executable not found: %w", err)
	}
	if strings.TrimSpace(root) == "" {
		return "", "", errors.New("Git workspace root is empty")
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	gitEntry := filepath.Join(absoluteRoot, ".git")
	if _, err := os.Stat(gitEntry); err != nil {
		return "", "", fmt.Errorf("project root %q is not a Git working tree", absoluteRoot)
	}

	output, err := runGit(gitPath, absoluteRoot, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", err
	}
	repoRoot = filepath.Clean(strings.TrimSpace(string(output)))
	absResolved, absErr := filepath.EvalSymlinks(absoluteRoot)
	repoResolved, repoErr := filepath.EvalSymlinks(repoRoot)
	if absErr == nil && repoErr == nil {
		absoluteRoot = filepath.Clean(absResolved)
		repoRoot = filepath.Clean(repoResolved)
	}
	if !samePath(absoluteRoot, repoRoot) {
		return "", "", fmt.Errorf("project root %q is not the Git root %q", absoluteRoot, repoRoot)
	}
	return gitPath, repoRoot, nil
}

func samePath(left, right string) bool {
	if os.PathSeparator == '\\' {
		left = strings.ToLower(left)
		right = strings.ToLower(right)
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func validateRelativePath(path string) (string, error) {
	if path == "" || filepath.IsAbs(path) {
		return "", errors.New("file path must be relative to the Git workspace")
	}

	cleanPath := filepath.Clean(filepath.FromSlash(path))
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", errors.New("file path must stay inside the Git workspace")
	}
	return cleanPath, nil
}

func runGit(gitPath, root string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	commandArgs := append([]string{"-C", root}, args...)
	cmd := exec.CommandContext(ctx, gitPath, commandArgs...)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return output, nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, errors.New("git command timed out")
	}
	message := strings.TrimSpace(string(output))
	if message == "" {
		message = err.Error()
	}
	return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), message)
}

func parseStatus(output []byte) (Status, error) {
	result := Status{Files: make([]FileStatus, 0)}
	entries := bytes.Split(output, []byte{0})
	for index := 0; index < len(entries); index++ {
		entry := entries[index]
		if len(entry) == 0 {
			continue
		}
		if bytes.HasPrefix(entry, []byte("## ")) {
			parseBranchHeader(&result, string(entry[3:]))
			continue
		}
		if len(entry) < 4 || entry[2] != ' ' {
			return Status{}, fmt.Errorf("unrecognized git status record %q", entry)
		}

		indexCode := entry[0]
		worktreeCode := entry[1]
		file := FileStatus{
			Code:      string([]byte{indexCode, worktreeCode}),
			Path:      string(entry[3:]),
			Untracked: indexCode == '?' && worktreeCode == '?',
			Staged:    indexCode != ' ' && indexCode != '?',
			Unstaged:  worktreeCode != ' ' && indexCode != '?',
			Conflict:  indexCode == 'U' || worktreeCode == 'U' || string([]byte{indexCode, worktreeCode}) == "AA" || string([]byte{indexCode, worktreeCode}) == "DD",
		}
		if indexCode == 'R' || indexCode == 'C' || worktreeCode == 'R' || worktreeCode == 'C' {
			if index+1 >= len(entries) {
				return Status{}, fmt.Errorf("missing original path for git status record %q", entry)
			}
			index++
			file.OriginalPath = string(entries[index])
		}
		result.Files = append(result.Files, file)
	}
	sort.SliceStable(result.Files, func(i, j int) bool {
		return result.Files[i].Path < result.Files[j].Path
	})
	return result, nil
}

func parseBranchHeader(result *Status, header string) {
	branch := strings.TrimSpace(strings.SplitN(header, "...", 2)[0])
	branch = strings.TrimPrefix(branch, "No commits yet on ")
	if branch == "HEAD (no branch)" {
		branch = "(detached HEAD)"
	}
	result.Branch = branch

	match := aheadBehindPattern.FindStringSubmatch(header)
	if len(match) == 5 {
		if match[1] != "" {
			result.Ahead, _ = strconv.Atoi(match[1])
			result.Behind, _ = strconv.Atoi(match[2])
		} else {
			result.Behind, _ = strconv.Atoi(match[3])
			result.Ahead, _ = strconv.Atoi(match[4])
		}
	}
}

func isMissingGitObject(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "does not exist in") ||
		strings.Contains(message, "exists on disk, but not in") ||
		strings.Contains(message, "invalid object name") ||
		strings.Contains(message, "needed a single revision") ||
		strings.Contains(message, "path '") && strings.Contains(message, "does not exist")
}
