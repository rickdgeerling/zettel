package store

import (
	"fmt"
	"os"
	"path/filepath"
)

type Store struct {
	Root     string
	cardsDir string
	archDir  string
}

func Init(root string) (*Store, error) {
	cardsDir := filepath.Join(root, "cards")
	archivedDir := filepath.Join(root, "archived")

	for _, dir := range []string{cardsDir, archivedDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	return &Store{
		Root:     root,
		cardsDir: cardsDir,
		archDir:  archivedDir,
	}, nil
}

func DefaultStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find home directory: %w", err)
	}
	return Init(filepath.Join(home, ".zettel"))
}
