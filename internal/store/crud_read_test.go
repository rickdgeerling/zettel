package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadCard(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{
		Slug:     "test-read",
		Title:    "Test Read Card",
		Created:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Modified: time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
		Source:   "cli",
		Tags:     []string{"testing"},
		Category: "backend",
		Body:     "Card body content",
	}
	path := filepath.Join(s.cardsDir, "test-read.md")
	_ = os.WriteFile(path, []byte(card.Marshal()), 0644)

	result, err := s.ReadCard("test-read")
	if err != nil {
		t.Fatalf("ReadCard failed: %v", err)
	}
	if result.Slug != "test-read" {
		t.Errorf("Slug mismatch: got %q, want %q", result.Slug, "test-read")
	}
	if result.Title != "Test Read Card" {
		t.Errorf("Title mismatch: got %q, want %q", result.Title, "Test Read Card")
	}
	if result.Body != "Card body content" {
		t.Errorf("Body mismatch: got %q, want %q", result.Body, "Card body content")
	}
}

func TestReadCardFromArchived(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{
		Slug:     "archived-card",
		Title:    "Archived Card",
		Created:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Modified: time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
		Source:   "cli",
		Body:     "Archived content",
	}
	path := filepath.Join(s.archDir, "archived-card.md")
	_ = os.WriteFile(path, []byte(card.Marshal()), 0644)

	result, err := s.ReadCard("archived-card")
	if err != nil {
		t.Fatalf("ReadCard failed for archived card: %v", err)
	}
	if result.Title != "Archived Card" {
		t.Errorf("Title mismatch: got %q, want %q", result.Title, "Archived Card")
	}
}

func TestReadCardNotFound(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	_, err := s.ReadCard("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent card, got nil")
	}
	if err != ErrCardNotFound {
		t.Errorf("Expected ErrCardNotFound, got %v", err)
	}
}
