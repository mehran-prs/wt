// Package store persists the alias -> repo cache on disk as JSON.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Repo struct {
	Alias  string `json:"alias"`
	Source string `json:"source"` // path or URL the user gave us
	Clone  string `json:"clone"`  // where we keep the local clone
}

type Store struct {
	path  string
	Repos map[string]Repo `json:"repos"`
}

func dir() (string, error) {
	if d := os.Getenv("WT_CACHE_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".wt"), nil
}

// ReposDir is where cloned repos live, one subfolder per alias.
func ReposDir() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "repos"), nil
}

func Load() (*Store, error) {
	d, err := dir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(d, "aliases.json")
	s := &Store{path: path, Repos: map[string]Repo{}}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	s.path = path
	return s, nil
}

func (s *Store) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func (s *Store) Aliases() []string {
	names := make([]string, 0, len(s.Repos))
	for a := range s.Repos {
		names = append(names, a)
	}
	return names
}
