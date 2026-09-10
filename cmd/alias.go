package cmd

import (
	"fmt"
	"os"

	"github.com/mehran-prs/wt/internal/store"
	"github.com/spf13/cobra"
)

var aliasCmd = &cobra.Command{
	Use:   "alias",
	Short: "Manage cached repo aliases",
}

var aliasRemoveCmd = &cobra.Command{
	Use:   "remove {alias}",
	Short: "Forget a cached alias (existing worktrees are left untouched)",
	Args:  cobra.ExactArgs(1),
	RunE:  runAliasRemove,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		s, err := store.Load()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return s.Aliases(), cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	aliasCmd.AddCommand(aliasRemoveCmd)
	rootCmd.AddCommand(aliasCmd)
}

func runAliasRemove(cmd *cobra.Command, args []string) error {
	alias := args[0]

	s, err := store.Load()
	if err != nil {
		return err
	}
	if _, ok := s.Repos[alias]; !ok {
		return fmt.Errorf("no such alias: %s", alias)
	}
	delete(s.Repos, alias)
	if err := s.Save(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Removed alias %q from cache.\n", alias)
	return nil
}
