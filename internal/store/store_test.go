package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreInit(t *testing.T) {
	tmp := t.TempDir()
	s, err := Init(tmp)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if s == nil {
		t.Fatal("Init returned nil Store")
	}

	cardsDir := filepath.Join(tmp, "cards")
	archivedDir := filepath.Join(tmp, "archived")
	if _, err := os.Stat(cardsDir); os.IsNotExist(err) {
		t.Errorf("cards directory not created: %v", err)
	}
	if _, err := os.Stat(archivedDir); os.IsNotExist(err) {
		t.Errorf("archived directory not created: %v", err)
	}
}

func TestStoreInitCreatesNested(t *testing.T) {
	tmp := t.TempDir()
	nested := filepath.Join(tmp, "subdir", "nested")
	s, err := Init(nested)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if s == nil {
		t.Fatal("Init returned nil Store")
	}

	cardsDir := filepath.Join(nested, "cards")
	archivedDir := filepath.Join(nested, "archived")
	if _, err := os.Stat(cardsDir); os.IsNotExist(err) {
		t.Errorf("cards directory not created: %v", err)
	}
	if _, err := os.Stat(archivedDir); os.IsNotExist(err) {
		t.Errorf("archived directory not created: %v", err)
	}
}

func TestStoreRootPath(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)
	if s.Root != tmp {
		t.Errorf("Store.Root = %q, want %q", s.Root, tmp)
	}
}
