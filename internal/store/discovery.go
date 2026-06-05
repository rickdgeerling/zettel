package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindStore walks up the directory tree from cwd looking for a .zettel/
// directory. It resolves symlinks on cwd first, then walks up to the
// filesystem root. Returns an error if no .zettel/ store is found or if
// any I/O error occurs (e.g., permission denied).
func FindStore(cwd string) (*Store, error) {
	resolved, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return nil, fmt.Errorf("resolving current directory: %w", err)
	}
	for dir := resolved; dir != "/"; dir = filepath.Dir(dir) {
		storeDir := filepath.Join(dir, ".zettel")
		if isValidStore(storeDir) {
			return Init(storeDir)
		}
		// Check root boundary
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return nil, fmt.Errorf("no .zettel store found (run 'zettel init' to create one in the current directory, set ZETTEL_HOME, or use --store)")
}

// ResolveStore determines the store path using the precedence:
// --store flag > walk up from CWD > ZETTEL_HOME env var.
// flagPath is the value of the --store CLI flag (empty if not set).
// envPath is the value of the ZETTEL_HOME environment variable (empty if not set).
func ResolveStore(cwd, flagPath, envPath string) (*Store, error) {
	if flagPath != "" {
		if !isValidStore(flagPath) {
			return nil, fmt.Errorf("store at %s is invalid: missing cards/ and/or archived/ subdirectories", flagPath)
		}
		return Init(flagPath)
	}

	store, err := FindStore(cwd)
	if err == nil {
		return store, nil
	}

	// Tree-walk failed — try envPath if set
	if envPath != "" {
		if !isValidStore(envPath) {
			return nil, fmt.Errorf("store at %s (from $ZETTEL_HOME) is invalid: missing cards/ and/or archived/ subdirectories", envPath)
		}
		return Init(envPath)
	}

	// No store found anywhere, return the tree-walk error
	return nil, err
}

// isValidStore checks whether root contains the expected cards/ and archived/
// subdirectories. Returns false if either directory is missing or inaccessible.
func isValidStore(root string) bool {
	cardsDir := filepath.Join(root, CardsDirName)
	if info, err := os.Stat(cardsDir); err != nil || !info.IsDir() {
		return false
	}

	archivedDir := filepath.Join(root, ArchivedDirName)
	if info, err := os.Stat(archivedDir); err != nil || !info.IsDir() {
		return false
	}

	return true
}
