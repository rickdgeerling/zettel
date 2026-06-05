package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteCardNew(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{
		Slug:    "new-card",
		Title:   "New Card Title",
		Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Body:    "Card body content",
	}

	err := s.WriteCard("new-card", card, "cli")
	if err != nil {
		t.Fatalf("WriteCard failed: %v", err)
	}

	path := filepath.Join(s.cardsDir, "new-card.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Card file not created: %v", err)
	}

	parsed, err := UnmarshalCard("new-card", data)
	if err != nil {
		t.Fatalf("Card not parseable: %v", err)
	}
	if parsed.Title != "New Card Title" {
		t.Errorf("Title mismatch: got %q", parsed.Title)
	}
	if parsed.Modified.IsZero() {
		t.Error("Modified not auto-set")
	}
	if parsed.Source != "cli" {
		t.Errorf("Source not auto-set: got %q", parsed.Source)
	}
}

func TestWriteCardOverwrite(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card1 := &Card{
		Slug:    "overwrite-me",
		Title:   "Original Title",
		Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Body:    "Original body",
	}
	_ = s.WriteCard("overwrite-me", card1, "cli")

	card2 := &Card{
		Slug:    "overwrite-me",
		Title:   "Updated Title",
		Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Body:    "Updated body",
	}
	err := s.WriteCard("overwrite-me", card2, "mcp")
	if err != nil {
		t.Fatalf("WriteCard overwrite failed: %v", err)
	}

	parsed, _ := s.ReadCard("overwrite-me")
	if parsed.Title != "Updated Title" {
		t.Errorf("Title not updated: got %q", parsed.Title)
	}
	if parsed.Source != "mcp" {
		t.Errorf("Source not updated: got %q", parsed.Source)
	}
}

func TestWriteCardInvalidSlug(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{
		Slug:    "Invalid-Slug",
		Title:   "Test",
		Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Body:    "Body",
	}

	err := s.WriteCard("Invalid-Slug", card, "cli")
	if err == nil {
		t.Error("Expected error for invalid slug, got nil")
	}
}

func TestWriteCardMissingTitle(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{
		Slug:    "missing-title",
		Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Body:    "Body",
	}

	err := s.WriteCard("missing-title", card, "cli")
	if err == nil {
		t.Error("Expected error for missing title, got nil")
	}
}

func TestWriteCardMissingCreated(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{
		Slug:  "missing-created",
		Title: "Test Title",
		Body:  "Body",
	}

	err := s.WriteCard("missing-created", card, "cli")
	if err == nil {
		t.Error("Expected error for missing created, got nil")
	}
}

func TestWriteCardPreservesCallerProvidedModified(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	callerProvided := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	card := &Card{
		Slug:     "preserve-modified",
		Title:    "Test",
		Created:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Modified: callerProvided,
		Body:     "Body",
	}

	_ = s.WriteCard("preserve-modified", card, "cli")
	parsed, _ := s.ReadCard("preserve-modified")

	if !parsed.Modified.Equal(callerProvided) {
		t.Errorf("Modified was overwritten: got %v, want %v", parsed.Modified, callerProvided)
	}
}
