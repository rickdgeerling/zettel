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

func TestArchiveCardAlreadyArchivedIsIdempotent(t *testing.T) {
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

	// Archiving an already-archived card is a successful no-op.
	if err := s.ArchiveCard("already-archived"); err != nil {
		t.Errorf("Expected idempotent success for already-archived card, got error: %v", err)
	}

	// The archived copy must remain.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Archived card disappeared after idempotent archive")
	}
}

func TestArchiveCardTwice(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{
		Slug:    "twice",
		Title:   "Archive Twice",
		Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Body:    "Content",
	}
	_ = s.WriteCard("twice", card, "test")

	if err := s.ArchiveCard("twice"); err != nil {
		t.Fatalf("First archive failed: %v", err)
	}
	if err := s.ArchiveCard("twice"); err != nil {
		t.Errorf("Second archive should be a no-op, got error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(s.cardsDir, "twice.md")); !os.IsNotExist(err) {
		t.Error("Card still in active directory after two archives")
	}
	if _, err := os.Stat(filepath.Join(s.archDir, "twice.md")); os.IsNotExist(err) {
		t.Error("Card not in archived directory after two archives")
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
