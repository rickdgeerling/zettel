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
	if err != nil {
		t.Fatalf("WriteCard failed: %v", err)
	}

	parsed, _ := s.ReadCard("missing-created")
	if parsed.Created.IsZero() {
		t.Error("Created was not auto-set")
	}
}

func TestWriteCardOverwriteBumpsModified(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	original := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	card := &Card{
		Slug:     "bump-modified",
		Title:    "Test",
		Created:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Modified: original,
		Body:     "v1",
	}
	_ = s.WriteCard("bump-modified", card, "cli")

	// Caller preserves a stale stamp (as read output does); the store must
	// bump modified anyway or the card would never show up in a
	// --modified-since window.
	stale := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	update := &Card{
		Slug:     "bump-modified",
		Title:    "Test",
		Created:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Modified: stale,
		Body:     "v2",
	}
	if err := s.WriteCard("bump-modified", update, "cli"); err != nil {
		t.Fatalf("WriteCard overwrite failed: %v", err)
	}

	parsed, _ := s.ReadCard("bump-modified")
	if !parsed.Modified.After(original) {
		t.Errorf("Modified not bumped on overwrite: got %v, want later than %v", parsed.Modified, original)
	}
}

func TestWriteCardOverwritePreservesCreated(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	first := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	card := &Card{
		Slug:    "keep-created",
		Title:   "Test",
		Created: first,
		Body:    "v1",
	}
	_ = s.WriteCard("keep-created", card, "cli")

	// A caller submitting a different created must not reset provenance.
	update := &Card{
		Slug:    "keep-created",
		Title:   "Test",
		Created: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Body:    "v2",
	}
	if err := s.WriteCard("keep-created", update, "cli"); err != nil {
		t.Fatalf("WriteCard overwrite failed: %v", err)
	}

	parsed, _ := s.ReadCard("keep-created")
	if !parsed.Created.Equal(first) {
		t.Errorf("Created not preserved on overwrite: got %v, want %v", parsed.Created, first)
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

// TestReadWriteRoundTrip mirrors the shell pattern `zettel read s | zettel
// write s`: the read output must be a valid write input that preserves all
// card content while bumping modified.
func TestReadWriteRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{
		Slug:    "round-trip",
		Title:   "Round Trip",
		Created: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		Source:  "cli",
		Tags:    []string{"testing", "roundtrip"},
		Body:    "Body prose with a [[wikilink]].\n\n## Links\n\n- [[other]]: related\n",
	}
	if err := s.WriteCard("round-trip", card, "cli"); err != nil {
		t.Fatalf("WriteCard failed: %v", err)
	}

	before, err := s.ReadCard("round-trip")
	if err != nil {
		t.Fatalf("ReadCard failed: %v", err)
	}

	reparsed, err := UnmarshalCard("round-trip", []byte(before.Marshal()))
	if err != nil {
		t.Fatalf("Read output is not a valid write input: %v", err)
	}
	if err := s.WriteCard("round-trip", reparsed, "cli"); err != nil {
		t.Fatalf("WriteCard round-trip failed: %v", err)
	}

	after, _ := s.ReadCard("round-trip")
	if after.Title != before.Title {
		t.Errorf("Title changed: got %q, want %q", after.Title, before.Title)
	}
	if !after.Created.Equal(before.Created) {
		t.Errorf("Created changed: got %v, want %v", after.Created, before.Created)
	}
	if after.Source != before.Source {
		t.Errorf("Source changed: got %q, want %q", after.Source, before.Source)
	}
	if after.Body != before.Body {
		t.Errorf("Body changed: got %q, want %q", after.Body, before.Body)
	}
	if after.Modified.Before(before.Modified) {
		t.Errorf("Modified went backwards: got %v, want >= %v", after.Modified, before.Modified)
	}
}
