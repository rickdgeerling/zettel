package store

import (
	"strings"
	"testing"
	"time"
)

func TestCardFrontmatterRoundTrip(t *testing.T) {
	card := &Card{
		Slug:     "test-card",
		Title:    "Test Card Title",
		Created:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Modified: time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
		Source:   "cli",
		Tags:     []string{"testing", "example"},
		Category: "backend",
		Status:   "",
		Body:     "This is the card body with a [[wikilink]].",
	}

	content := card.Marshal()

	parsed, err := UnmarshalCard("test-card", []byte(content))
	if err != nil {
		t.Fatalf("UnmarshalCard failed: %v", err)
	}

	if parsed.Title != card.Title {
		t.Errorf("Title mismatch: got %q, want %q", parsed.Title, card.Title)
	}
	if parsed.Slug != card.Slug {
		t.Errorf("Slug mismatch: got %q, want %q", parsed.Slug, card.Slug)
	}
	if !parsed.Created.Equal(card.Created) {
		t.Errorf("Created mismatch: got %v, want %v", parsed.Created, card.Created)
	}
	if !parsed.Modified.Equal(card.Modified) {
		t.Errorf("Modified mismatch: got %v, want %v", parsed.Modified, card.Modified)
	}
	if parsed.Source != card.Source {
		t.Errorf("Source mismatch: got %q, want %q", parsed.Source, card.Source)
	}
	if len(parsed.Tags) != len(card.Tags) {
		t.Errorf("Tags length mismatch: got %d, want %d", len(parsed.Tags), len(card.Tags))
	}
	if parsed.Category != card.Category {
		t.Errorf("Category mismatch: got %q, want %q", parsed.Category, card.Category)
	}
	if parsed.Body != card.Body {
		t.Errorf("Body mismatch: got %q, want %q", parsed.Body, card.Body)
	}
}

func TestUnmarshalCardInvalidYAML(t *testing.T) {
	_, err := UnmarshalCard("bad-card", []byte("not valid yaml ---\ntitle: test"))
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestSlugValidation(t *testing.T) {
	tests := []struct {
		slug  string
		valid bool
	}{
		{"valid-slug", true},
		{"abc", true},
		{"a", false},
		{"Abc", false},
		{"valid_slug", false},
		{"valid.Slug", false},
		{"very-long-slug-that-is-over-sixty-characters-in-length-but-under-60", false},
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			err := validateSlug(tt.slug)
			if (err == nil) != tt.valid {
				t.Errorf("validateSlug(%q) = %v, want valid=%v", tt.slug, err, tt.valid)
			}
		})
	}
}

func TestExtractWikilinks(t *testing.T) {
	body := "This links to [[jwt-revocation]] and [[oauth2-pkce]]. Also [[ab]] but not [[bad link]]."
	links := ExtractWikilinks(body)

	if len(links) != 3 {
		t.Errorf("Expected 3 links, got %d: %v", len(links), links)
	}
	if links[0] != "jwt-revocation" || links[1] != "oauth2-pkce" || links[2] != "ab" {
		t.Errorf("Unexpected link order: %v", links)
	}
}

func TestUnmarshalCardNoFrontmatter(t *testing.T) {
	_, err := UnmarshalCard("no-fm", []byte("just plain text without frontmatter"))
	if err == nil {
		t.Error("Expected error for missing frontmatter, got nil")
	}
}

func TestUnmarshalCardEmptyBody(t *testing.T) {
	card, err := UnmarshalCard("empty-body", []byte("title: Test\n---\n"))
	if err != nil {
		t.Fatalf("UnmarshalCard failed for empty body: %v", err)
	}
	if card.Body != "" {
		t.Errorf("Body = %q, want empty string", card.Body)
	}
}

func TestWikilinkExtractionDeduplication(t *testing.T) {
	body := "[[foo]] and [[foo]] again and [[bar]]"
	links := ExtractWikilinks(body)
	if len(links) != 2 {
		t.Errorf("Expected 2 unique links, got %d: %v", len(links), links)
	}
}

func TestSlugValidationEdgeCases(t *testing.T) {
	invalid := []string{
		"",
		"-starts-with-hyphen",
		"ends-with-hyphen-",
		"has space",
		"has\ttab",
		"has\nnewline",
		"UPPERCASE",
		"mixed-Case",
		"has_underscore",
		"has.dot",
		"has!bang",
	}
	for _, s := range invalid {
		err := validateSlug(s)
		if err == nil {
			t.Errorf("validateSlug(%q) = nil, want error", s)
		}
	}

	valid := []string{
		"ab",
		"abc",
		"a1",
		"a-b",
		"a-b-c",
		"abc123",
		"abc-123",
		"a0",
		"very-long-slug-that-is-over-sixty-characters-in-length",
	}
	for _, s := range valid {
		err := validateSlug(s)
		if err != nil {
			t.Errorf("validateSlug(%q) = %v, want nil", s, err)
		}
	}
}

func TestMarshalEmitsStandardDelimiters(t *testing.T) {
	card := &Card{
		Slug:    "delimiters",
		Title:   "Delimiters",
		Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Body:    "Body.",
	}

	content := card.Marshal()
	if !strings.HasPrefix(content, "---\n") {
		t.Errorf("Marshal output lacks leading --- delimiter:\n%q", content)
	}
	if _, err := UnmarshalCard("delimiters", []byte(content)); err != nil {
		t.Errorf("Marshal output is not parseable: %v", err)
	}
}

// TestMarshalRoundTripStable guards the `zettel read s | zettel write s`
// pattern: parse(Marshal(c)) must re-marshal byte-identically, so a
// round-trip never perturbs content the agent did not touch.
func TestMarshalRoundTripStable(t *testing.T) {
	card := &Card{
		Slug:     "stable",
		Title:    "Stable Card",
		Created:  time.Date(2025, 1, 15, 12, 30, 0, 0, time.UTC),
		Modified: time.Date(2025, 1, 20, 8, 0, 0, 0, time.UTC),
		Source:   "cli",
		Tags:     []string{"a", "b"},
		Category: "backend",
		Body:     "Prose.\n\n## Links\n\n- [[other]]: related\n\nA horizontal rule below.\n\n---\n\nStill body.\n",
	}

	once := card.Marshal()
	parsed, err := UnmarshalCard("stable", []byte(once))
	if err != nil {
		t.Fatalf("UnmarshalCard failed: %v", err)
	}
	if got := parsed.Marshal(); got != once {
		t.Errorf("Marshal not stable across parse:\ngot:  %q\nwant: %q", got, once)
	}
	if parsed.Body != card.Body {
		t.Errorf("Body not preserved: got %q, want %q", parsed.Body, card.Body)
	}
}

// TestUnmarshalCardLegacyFormat: cards written before the standard
// delimiters lack the leading ---. They must stay readable (and normalize
// on the next write).
func TestUnmarshalCardLegacyFormat(t *testing.T) {
	legacy := "title: Legacy\ncreated: 2025-01-15T00:00:00Z\nsource: cli\n---\nLegacy body.\n"

	parsed, err := UnmarshalCard("legacy", []byte(legacy))
	if err != nil {
		t.Fatalf("legacy card not parseable: %v", err)
	}
	if parsed.Title != "Legacy" {
		t.Errorf("Title = %q, want %q", parsed.Title, "Legacy")
	}
	if parsed.Body != "Legacy body.\n" {
		t.Errorf("Body = %q, want %q", parsed.Body, "Legacy body.\n")
	}
}

func TestCardDescriptionRoundTrip(t *testing.T) {
	card := &Card{
		Slug:        "with-desc",
		Title:       "With Description",
		Created:     time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Description: "What this card is about, in one line.",
		Body:        "Body.",
	}

	content := card.Marshal()
	parsed, err := UnmarshalCard("with-desc", []byte(content))
	if err != nil {
		t.Fatalf("UnmarshalCard failed: %v", err)
	}
	if parsed.Description != card.Description {
		t.Errorf("Description mismatch: got %q, want %q", parsed.Description, card.Description)
	}
	if got := parsed.Marshal(); got != content {
		t.Errorf("Marshal not stable with description:\ngot:  %q\nwant: %q", got, content)
	}

	// Absent description must stay absent (omitempty), not emit an empty field.
	without := &Card{Slug: "no-desc", Title: "No Description", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Body: "Body."}
	if got := without.Marshal(); strings.Contains(got, "description:") {
		t.Errorf("empty description emitted:\n%s", got)
	}
}
