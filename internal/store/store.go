package store

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	CardsDirName    = "cards"
	ArchivedDirName = "archived"
)

type Store struct {
	Root     string
	cardsDir string
	archDir  string
}

func Init(root string) (*Store, error) {
	cardsDir := filepath.Join(root, CardsDirName)
	archivedDir := filepath.Join(root, ArchivedDirName)

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
