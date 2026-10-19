package store

import (
	"testing"
	"time"
)

func TestAppendToSectionExisting(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{
		Slug:    "append-me",
		Title:   "Append Me",
		Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Body:    "Prose paragraph.\n\n## Links\n\n- [[a]]: first\n",
	}
	_ = s.WriteCard("append-me", card, "test")

	if err := s.AppendToSection("append-me", "Links", "- [[b]]: second"); err != nil {
		t.Fatalf("AppendToSection failed: %v", err)
	}

	parsed, _ := s.ReadCard("append-me")
	want := "Prose paragraph.\n\n## Links\n\n- [[a]]: first\n- [[b]]: second\n"
	if parsed.Body != want {
		t.Errorf("Body = %q, want %q", parsed.Body, want)
	}
}

func TestAppendToSectionCreatesSection(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{
		Slug:    "no-section",
		Title:   "No Section",
		Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Body:    "Just prose.",
	}
	_ = s.WriteCard("no-section", card, "test")

	if err := s.AppendToSection("no-section", "Links", "- [[x]]: related"); err != nil {
		t.Fatalf("AppendToSection failed: %v", err)
	}

	parsed, _ := s.ReadCard("no-section")
	want := "Just prose.\n\n## Links\n\n- [[x]]: related\n"
	if parsed.Body != want {
		t.Errorf("Body = %q, want %q", parsed.Body, want)
	}
}

func TestAppendToSectionBeforeNextHeading(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{
		Slug:    "two-sections",
		Title:   "Two Sections",
		Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Body:    "## Links\n\n- [[a]]: x\n\n## Notes\n\nnote text",
	}
	_ = s.WriteCard("two-sections", card, "test")

	if err := s.AppendToSection("two-sections", "Links", "- [[b]]: y"); err != nil {
		t.Fatalf("AppendToSection failed: %v", err)
	}

	parsed, _ := s.ReadCard("two-sections")
	want := "## Links\n\n- [[a]]: x\n- [[b]]: y\n\n## Notes\n\nnote text\n"
	if parsed.Body != want {
		t.Errorf("Body = %q, want %q", parsed.Body, want)
	}
}

func TestAppendToSectionHeadingOnly(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{
		Slug:    "empty-section",
		Title:   "Empty Section",
		Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Body:    "## Links\n",
	}
	_ = s.WriteCard("empty-section", card, "test")

	if err := s.AppendToSection("empty-section", "Links", "- [[x]]: first"); err != nil {
		t.Fatalf("AppendToSection failed: %v", err)
	}

	parsed, _ := s.ReadCard("empty-section")
	want := "## Links\n\n- [[x]]: first\n"
	if parsed.Body != want {
		t.Errorf("Body = %q, want %q", parsed.Body, want)
	}
}

func TestAppendToSectionNormalizesHeadingArg(t *testing.T) {
	body := appendLineToSection("Prose.\n", "## Links", "- [[x]]: y")
	want := "Prose.\n\n## Links\n\n- [[x]]: y\n"
	if body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestAppendToSectionMissingCard(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	if err := s.AppendToSection("nonexistent", "Links", "- [[x]]: y"); err == nil {
		t.Error("Expected error for missing card, got nil")
	}
}

func TestAppendToSectionBumpsModified(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	original := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	card := &Card{
		Slug:     "bump",
		Title:    "Bump",
		Created:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Modified: original,
		Body:     "## Links\n\n- [[a]]: x\n",
	}
	_ = s.WriteCard("bump", card, "test")

	if err := s.AppendToSection("bump", "Links", "- [[b]]: y"); err != nil {
		t.Fatalf("AppendToSection failed: %v", err)
	}

	parsed, _ := s.ReadCard("bump")
	if !parsed.Modified.After(original) {
		t.Errorf("Modified not bumped: got %v, want later than %v", parsed.Modified, original)
	}
}
