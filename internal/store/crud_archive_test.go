package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestArchiveCard(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{
		Slug:    "to-archive",
		Title:   "Will Be Archived",
		Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Body:    "Content",
	}
	_ = s.WriteCard("to-archive", card, "cli")

	err := s.ArchiveCard("to-archive")
	if err != nil {
		t.Fatalf("ArchiveCard failed: %v", err)
	}

	path := filepath.Join(s.cardsDir, "to-archive.md")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("Card still exists in active directory after archive")
	}

	path = filepath.Join(s.archDir, "to-archive.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Card not found in archived directory")
	}
}

func TestArchiveCardNotFound(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	err := s.ArchiveCard("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent card, got nil")
	}
}

func TestArchiveCardAlreadyArchived(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{
		Slug:    "already-archived",
		Title:   "Already Archived",
		Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Body:    "Content",
	}
	path := filepath.Join(s.archDir, "already-archived.md")
	_ = os.WriteFile(path, []byte(card.Marshal()), 0644)

	err := s.ArchiveCard("already-archived")
	if err == nil {
		t.Error("Expected error for already-archived card, got nil")
	}
}

func TestReadCardFromActiveAfterArchive(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{
		Slug:    "archive-test",
		Title:   "Archive Test",
		Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Body:    "Content",
	}
	_ = s.WriteCard("archive-test", card, "cli")
	_ = s.ArchiveCard("archive-test")

	result, err := s.ReadCard("archive-test")
	if err != nil {
		t.Fatalf("ReadCard failed for archived card: %v", err)
	}
	if result.Title != "Archive Test" {
		t.Errorf("Title mismatch: got %q", result.Title)
	}
}
