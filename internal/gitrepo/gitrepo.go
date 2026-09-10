// Package gitrepo wraps the git CLI operations wt needs: clone, pull, and
// worktree creation.
package gitrepo

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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

// AddWorktree creates a worktree at path for branch. If the branch doesn't
// exist yet, it is created from primary without tracking it as upstream.
func AddWorktree(repoDir, path, branch, primary string) error {
	if BranchExists(repoDir, branch) {
		_, err := run(repoDir, "worktree", "add", path, branch)
		return err
	}
	_, err := run(repoDir, "worktree", "add", "-b", branch, "--no-track", path, primary)
	return err
}
