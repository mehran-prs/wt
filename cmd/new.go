package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mehran-prs/wt/internal/gitrepo"
	"github.com/mehran-prs/wt/internal/store"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new {repo-or-alias} {branch}",
	Short: "Create (or switch to) a worktree for a branch",
	Args:  cobra.ExactArgs(2),
	RunE:  runNew,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		s, err := store.Load()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		switch len(args) {
		case 0:
			return s.Aliases(), cobra.ShellCompDirectiveNoFileComp
		case 1:
			repo, ok := s.Repos[args[0]]
			if !ok {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			branches, err := gitrepo.ListBranches(repo.Clone)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return branches, cobra.ShellCompDirectiveNoFileComp
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
}

func worktreeBaseDir() (string, error) {
	if d := os.Getenv("WT_WORKTREE_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "wt"), nil
}

func looksLikePath(s string) bool {
	return strings.ContainsAny(s, "/:") || strings.HasSuffix(s, ".git")
}

// isSCPLike matches git's scp-style remotes, e.g. git@github.com:org/repo.git.
func isSCPLike(s string) bool {
	at := strings.Index(s, "@")
	colon := strings.Index(s, ":")
	return at >= 0 && colon > at
}

// normalizeSource turns a bare "host/org/repo" shorthand (e.g.
// github.com/mehran-prs/snip) into a full https clone URL. Local paths,
// scp-style remotes, and URLs that already have a scheme are left untouched.
func normalizeSource(s string) string {
	if strings.Contains(s, "://") || isSCPLike(s) {
		return s
	}
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") || strings.HasPrefix(s, "~") {
		return s
	}
	if _, err := os.Stat(s); err == nil {
		return s
	}
	if !strings.HasSuffix(s, ".git") {
		s += ".git"
	}
	return "https://" + s
}

func prompt(msg string) (string, error) {
	fmt.Fprint(os.Stderr, msg)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func promptForSource(alias string) (string, error) {
	line, err := prompt(fmt.Sprintf("No cached repo for alias %q. Enter the repo path/URL to clone: ", alias))
	if err != nil {
		return "", err
	}
	if line == "" {
		return "", fmt.Errorf("no repo path entered")
	}
	return line, nil
}

func promptForAlias(source string) (string, error) {
	line, err := prompt(fmt.Sprintf("No alias cached for %q. Enter an alias to remember it as: ", source))
	if err != nil {
		return "", err
	}
	if line == "" {
		return "", fmt.Errorf("no alias entered")
	}
	return line, nil
}

// resolveRepo figures out which alias/source we're working with, cloning it
// into the cache the first time it's seen.
func resolveRepo(s *store.Store, input string) (store.Repo, error) {
	reposDir, err := store.ReposDir()
	if err != nil {
		return store.Repo{}, err
	}

	if r, ok := s.Repos[input]; ok {
		return r, nil
	}

	var alias, source string
	if looksLikePath(input) {
		source = input
		alias, err = promptForAlias(source)
		if err != nil {
			return store.Repo{}, err
		}
		if r, ok := s.Repos[alias]; ok {
			return r, nil
		}
	} else {
		alias = input
		source, err = promptForSource(alias)
		if err != nil {
			return store.Repo{}, err
		}
	}

	source = normalizeSource(source)
	repo := store.Repo{
		Alias:  alias,
		Source: source,
		Clone:  filepath.Join(reposDir, alias),
	}
	fmt.Fprintf(os.Stderr, "Cloning %s into cache (alias %q)...\n", source, alias)
	if err := gitrepo.Clone(source, repo.Clone); err != nil {
		return store.Repo{}, err
	}
	s.Repos[alias] = repo
	if err := s.Save(); err != nil {
		return store.Repo{}, err
	}
	return repo, nil
}

func runNew(cmd *cobra.Command, args []string) error {
	input, branch := args[0], args[1]

	s, err := store.Load()
	if err != nil {
		return err
	}

	repo, err := resolveRepo(s, input)
	if err != nil {
		return err
	}

	baseDir, err := worktreeBaseDir()
	if err != nil {
		return err
	}
	worktreePath := filepath.Join(baseDir, repo.Alias+"-"+branch)

	got, exists, err := gitrepo.ExistingWorktree(repo.Clone, worktreePath)
	if err != nil {
		return err
	}
	if exists {
		if got != branch {
			return fmt.Errorf("%s already exists as a worktree on branch %q, not %q", worktreePath, got, branch)
		}
		fmt.Println(worktreePath)
		return nil
	}

	fmt.Fprintln(os.Stderr, "Detecting primary branch...")
	primary, err := gitrepo.PrimaryBranch(repo.Clone)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Pulling latest %s...\n", primary)
	if err := gitrepo.UpdatePrimary(repo.Clone, primary); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Creating worktree at %s...\n", worktreePath)
	if err := gitrepo.AddWorktree(repo.Clone, worktreePath, branch, primary); err != nil {
		return err
	}

	fmt.Println(worktreePath)
	return nil
}
