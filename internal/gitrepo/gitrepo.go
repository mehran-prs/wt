// Package gitrepo wraps the git CLI operations wt needs: clone, pull, and
// worktree creation.
package gitrepo

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, out.String())
	}
	return strings.TrimSpace(out.String()), nil
}

// Clone clones source into dest if dest doesn't already exist.
func Clone(source, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return nil
	}
	_, err := run("", "clone", source, dest)
	return err
}

// PrimaryBranch returns "master" if origin/master exists, else "main".
func PrimaryBranch(repoDir string) (string, error) {
	if _, err := run(repoDir, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/master"); err == nil {
		return "master", nil
	}
	if _, err := run(repoDir, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/main"); err == nil {
		return "main", nil
	}
	return "", fmt.Errorf("could not detect primary branch (neither origin/master nor origin/main exists)")
}

// UpdatePrimary fetches and fast-forwards the primary branch.
func UpdatePrimary(repoDir, primary string) error {
	if _, err := run(repoDir, "fetch", "origin"); err != nil {
		return err
	}
	if _, err := run(repoDir, "checkout", primary); err != nil {
		return err
	}
	_, err := run(repoDir, "pull", "--ff-only", "origin", primary)
	return err
}

// BranchExists reports whether branch exists locally or on origin.
func BranchExists(repoDir, branch string) bool {
	if _, err := run(repoDir, "show-ref", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		return true
	}
	if _, err := run(repoDir, "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+branch); err == nil {
		return true
	}
	return false
}

// ExistingWorktree reports the branch path is already checked out to, if
// path exists and is a registered worktree of repoDir. If path doesn't
// exist at all, exists is false and err is nil.
func ExistingWorktree(repoDir, path string) (branch string, exists bool, err error) {
	if _, err := os.Stat(path); err != nil {
		return "", false, nil
	}
	out, err := run(repoDir, "worktree", "list", "--porcelain")
	if err != nil {
		return "", false, err
	}
	abs, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", false, err
	}
	for block := range strings.SplitSeq(out, "\n\n") {
		var wtPath, wtBranch string
		for line := range strings.SplitSeq(block, "\n") {
			if p, ok := strings.CutPrefix(line, "worktree "); ok {
				wtPath = p
			} else if b, ok := strings.CutPrefix(line, "branch "); ok {
				wtBranch = strings.TrimPrefix(b, "refs/heads/")
			}
		}
		if wtPath == "" {
			continue
		}
		if resolved, err := filepath.EvalSymlinks(wtPath); err == nil && resolved == abs {
			return wtBranch, true, nil
		}
	}
	return "", true, fmt.Errorf("%s already exists and is not a worktree of this repo", path)
}

// AddWorktree creates a worktree at path for branch. If the branch doesn't
// exist yet, it is created from primary without tracking it as upstream.
// Callers should check ExistingWorktree first; path is assumed not to exist.
func AddWorktree(repoDir, path, branch, primary string) error {
	if BranchExists(repoDir, branch) {
		_, err := run(repoDir, "worktree", "add", path, branch)
		return err
	}
	_, err := run(repoDir, "worktree", "add", "-b", branch, "--no-track", path, primary)
	return err
}

// ListBranches returns local and remote-tracking branch names (deduped,
// without the "origin/" prefix or the origin/HEAD pointer).
func ListBranches(repoDir string) ([]string, error) {
	out, err := run(repoDir, "for-each-ref", "--format=%(refname:short)", "refs/heads", "refs/remotes/origin")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var branches []string
	for line := range strings.SplitSeq(out, "\n") {
		name := strings.TrimPrefix(strings.TrimSpace(line), "origin/")
		if name == "" || name == "HEAD" || name == "origin" || seen[name] {
			continue
		}
		seen[name] = true
		branches = append(branches, name)
	}
	return branches, nil
}
