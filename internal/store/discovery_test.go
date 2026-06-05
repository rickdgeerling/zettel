package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsValidStore(t *testing.T) {
	t.Run("valid store", func(t *testing.T) {
		tmp := t.TempDir()
		s, err := Init(tmp)
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if !isValidStore(s.Root) {
			t.Error("isValidStore returned false for a valid store")
		}
	})

	t.Run("missing cards dir", func(t *testing.T) {
		tmp := t.TempDir()
		// Only create archived dir
		if err := os.MkdirAll(filepath.Join(tmp, ArchivedDirName), 0755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		if isValidStore(tmp) {
			t.Error("isValidStore returned true when cards/ is missing")
		}
	})

	t.Run("missing archived dir", func(t *testing.T) {
		tmp := t.TempDir()
		// Only create cards dir
		if err := os.MkdirAll(filepath.Join(tmp, CardsDirName), 0755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		if isValidStore(tmp) {
			t.Error("isValidStore returned true when archived/ is missing")
		}
	})

	t.Run("file instead of cards dir", func(t *testing.T) {
		tmp := t.TempDir()
		// Create archived dir and a file named "cards"
		if err := os.MkdirAll(filepath.Join(tmp, ArchivedDirName), 0755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmp, CardsDirName), nil, 0644); err != nil {
			t.Fatalf("writefile failed: %v", err)
		}
		if isValidStore(tmp) {
			t.Error("isValidStore returned true when cards is a file, not a dir")
		}
	})

	t.Run("nonexistent path", func(t *testing.T) {
		tmp := t.TempDir()
		nonexistent := filepath.Join(tmp, "doesnotexist")
		if isValidStore(nonexistent) {
			t.Error("isValidStore returned true for nonexistent path")
		}
	})
}

func TestFindStore(t *testing.T) {
	t.Run("finds store in current dir", func(t *testing.T) {
		tmp := t.TempDir()
		cwd := filepath.Join(tmp, "project")

		// Create project/.zettel as a valid store
		zettelDir := filepath.Join(cwd, ".zettel")
		s, err := Init(zettelDir)
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		found, err := FindStore(cwd)
		if err != nil {
			t.Fatalf("FindStore failed: %v", err)
		}
		if found.Root != s.Root {
			t.Errorf("FindStore returned Root=%q, want %q", found.Root, s.Root)
		}
	})

	t.Run("finds store in ancestor dir", func(t *testing.T) {
		tmp := t.TempDir()

		// Create store in tmp/.zettel
		zettelDir := filepath.Join(tmp, ".zettel")
		s, err := Init(zettelDir)
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Navigate deep into the project
		deepCwd := filepath.Join(tmp, "src", "internal", "pkg")
		if err := os.MkdirAll(deepCwd, 0755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}

		found, err := FindStore(deepCwd)
		if err != nil {
			t.Fatalf("FindStore failed: %v", err)
		}
		if found.Root != s.Root {
			t.Errorf("FindStore returned Root=%q, want %q", found.Root, s.Root)
		}
	})

	t.Run("walks to root with no store", func(t *testing.T) {
		tmp := t.TempDir()

		// No .zettel anywhere under tmp — so walking up will hit the filesystem root
		// without finding one. The walk starts at tmp (which is under /tmp/...)
		// and walks to /. No store anywhere on this path.
		_, err := FindStore(tmp)
		if err == nil {
			t.Fatal("FindStore should fail when no .zettel found")
		}
		if !strings.Contains(err.Error(), "no .zettel store found") {
			t.Errorf("error message should mention 'no .zettel store found', got: %v", err)
		}
	})

	t.Run("returns error on inaccessible path", func(t *testing.T) {
		tmp := t.TempDir()
		noPerm := filepath.Join(tmp, "noperm")
		if err := os.Mkdir(noPerm, 0000); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		defer os.Chmod(noPerm, 0755)

		// Trying to walk from a dir we can't read should fail
		cwd := filepath.Join(noPerm, "subdir")
		_, err := FindStore(cwd)
		if err == nil {
			t.Fatal("FindStore should fail when cwd is inaccessible")
		}
	})

	t.Run("errors on EvalSymlinks failure", func(t *testing.T) {
		tmp := t.TempDir()
		brokenSymlink := filepath.Join(tmp, "broken")
		if err := os.Symlink("/nonexistent/path", brokenSymlink); err != nil {
			t.Fatalf("symlink failed: %v", err)
		}
		_, err := FindStore(brokenSymlink)
		if err == nil {
			t.Fatal("FindStore should fail when EvalSymlinks fails")
		}
	})
}

func TestResolveStore(t *testing.T) {
	t.Run("flag path takes precedence", func(t *testing.T) {
		tmp := t.TempDir()

		// Create a valid store for the flag path
		flagStore, err := Init(tmp)
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Create a separate store via tree-walk (should be ignored)
		cwd := filepath.Join(t.TempDir(), "project")
		zettelDir := filepath.Join(cwd, ".zettel")
		if _, err := Init(zettelDir); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		result, err := ResolveStore(cwd, tmp, "")
		if err != nil {
			t.Fatalf("ResolveStore failed: %v", err)
		}
		if result.Root != flagStore.Root {
			t.Errorf("ResolveStore returned Root=%q, want flagStore.Root=%q", result.Root, flagStore.Root)
		}
	})

	t.Run("tree-walk when no flag", func(t *testing.T) {
		tmp := t.TempDir()
		cwd := filepath.Join(tmp, "project")
		zettelDir := filepath.Join(cwd, ".zettel")
		s, err := Init(zettelDir)
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		result, err := ResolveStore(cwd, "", "")
		if err != nil {
			t.Fatalf("ResolveStore failed: %v", err)
		}
		if result.Root != s.Root {
			t.Errorf("ResolveStore returned Root=%q, want %q", result.Root, s.Root)
		}
	})

	t.Run("fallback to ZETTEL_HOME when tree-walk fails", func(t *testing.T) {
		tmp := t.TempDir()
		envStore, err := Init(tmp)
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// CWD has no .zettel anywhere
		empty := t.TempDir()

		result, err := ResolveStore(empty, "", tmp)
		if err != nil {
			t.Fatalf("ResolveStore should fall back to envPath: %v", err)
		}
		if result.Root != envStore.Root {
			t.Errorf("ResolveStore returned Root=%q, want envStore.Root=%q", result.Root, envStore.Root)
		}
	})

	t.Run("returns tree-walk error when no store found anywhere", func(t *testing.T) {
		tmp := t.TempDir()
		_, err := ResolveStore(tmp, "", "")
		if err == nil {
			t.Fatal("ResolveStore should fail when no store found")
		}
	})

	t.Run("invalid flag path error message", func(t *testing.T) {
		tmp := t.TempDir()
		// tmp has no cards/ or archived/ subdirs
		_, err := ResolveStore("/some/cwd", tmp, "")
		if err == nil {
			t.Fatal("ResolveStore should fail for invalid flag path")
		}
		expected := "store at " + tmp + " is invalid: missing cards/ and/or archived/ subdirectories"
		if err.Error() != expected {
			t.Errorf("error message mismatch:\n  got:  %q\n  want: %q", err.Error(), expected)
		}
	})

	t.Run("invalid env path error message", func(t *testing.T) {
		tmp := t.TempDir()
		empty := t.TempDir()
		_, err := ResolveStore(empty, "", tmp)
		if err == nil {
			t.Fatal("ResolveStore should fail for invalid env path")
		}
		expected := "store at " + tmp + " (from $ZETTEL_HOME) is invalid: missing cards/ and/or archived/ subdirectories"
		if err.Error() != expected {
			t.Errorf("error message mismatch:\n  got:  %q\n  want: %q", err.Error(), expected)
		}
	})
}
