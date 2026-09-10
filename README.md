# wt

Fast `git worktree` creation across all the repos you work on.

## Install

```sh
go install github.com/mehran-prs/wt@latest
```

## Usage

### Create (or switch to) a worktree

```sh
wt new <repo-or-alias> <branch>
```

```sh
wt new github.com/mehran-prs/snip my-feature   # first time: asks for an alias to save it under
wt new snip another-feature                    # later: reuse it by that alias
cd "$(wt new snip my-feature)"                 # only the worktree path goes to stdout, so this just works
```

Any repo location git understands works: a local path, a full URL, an ssh
remote, or a bare `host/org/repo` shorthand.

Details:
- The first time you use a repo, `wt` asks you to pick an alias for it and
  clones it into its cache. The next time, use that alias instead.
- If you type an alias that isn't cached yet, `wt` asks you to paste the
  repo path/URL instead, clones it, and remembers it under that alias.
- Before creating the worktree, `wt` fetches and fast-forwards the primary
  branch (`master` if it exists, otherwise `main`).
- Branch resolution:
  - exists locally already -> just checked out into the new worktree.
  - exists on `origin` but not locally -> checked out and tracks `origin/<branch>`.
  - doesn't exist anywhere -> created from the primary branch, **without**
    tracking it as upstream.
- The worktree is created at `$WT_WORKTREE_DIR/<alias>-<branch>` (default
  `~/wt/<alias>-<branch>`).


### Forget an alias

`wt` keeps a small local cache of repos you've used before, each under a
short alias, and reuses that cache to create new worktrees in seconds.

```sh
wt alias remove <alias>
```

Removes the alias from the cache. Any worktrees already created from it are
left untouched — that's on you to clean up.

## Configuration

| Env var            | Default   | Purpose                                  |
|---------------------|-----------|-------------------------------------------|
| `WT_WORKTREE_DIR`   | `~/wt`    | Where new worktrees are created           |
| `WT_CACHE_DIR`      | `~/.wt`   | Where cloned repos and aliases are stored |

## Shell completion

`wt` supports alias and command completion. Generate a script for your shell
and source it, e.g. for zsh:

```sh
wt completion zsh > "${fpath[1]}/_wt"
```

Run `wt completion --help` for bash/fish/powershell instructions.
