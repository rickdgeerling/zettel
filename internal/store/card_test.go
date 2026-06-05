package store

import (
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
