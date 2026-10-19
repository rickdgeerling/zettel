package store

import (
	"strings"
	"testing"
	"time"
)

func TestDraftIndex(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	cards := []*Card{
		{Slug: "queue-backpressure", Title: "Queue Backpressure", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Category: "backend", Body: "Content"},
		{Slug: "sql-migration-gotcha", Title: "SQL Migration Gotcha", Created: time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC), Category: "backend", Body: "Content"},
		{Slug: "react-key-pitfall", Title: "React Key Pitfall", Created: time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC), Category: "frontend", Body: "Content"},
		{Slug: "unfiled-insight", Title: "Unfiled Insight", Created: time.Date(2025, 1, 18, 0, 0, 0, 0, time.UTC), Body: "Content"}, // no category
	}
	for _, c := range cards {
		_ = s.WriteCard(c.Slug, c, "test")
	}

	// Pre-existing index card: creation date must survive redrafting.
	originalCreated := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	index := &Card{Slug: "index", Title: "Keyword Index", Created: originalCreated, Body: "# old index\n"}
	_ = s.WriteCard("index", index, "test")

	draft, err := s.DraftIndex()
	if err != nil {
		t.Fatalf("DraftIndex failed: %v", err)
	}

	// The draft must itself be a valid write input.
	parsed, err := UnmarshalCard("index", []byte(draft))
	if err != nil {
		t.Fatalf("draft not parseable: %v\n%s", err, draft)
	}
	if parsed.Title != "Keyword Index" {
		t.Errorf("Title = %q, want %q", parsed.Title, "Keyword Index")
	}
	if !parsed.Created.Equal(originalCreated) {
		t.Errorf("Created = %v, want preserved %v", parsed.Created, originalCreated)
	}

	// Categories are alphabetical, entries slug-sorted within them.
	wantBody := "## Uncategorized\n\n- [[unfiled-insight]] — Unfiled Insight\n\n" +
		"## backend\n\n- [[queue-backpressure]] — Queue Backpressure\n- [[sql-migration-gotcha]] — SQL Migration Gotcha\n\n" +
		"## frontend\n\n- [[react-key-pitfall]] — React Key Pitfall\n\n"
	if parsed.Body != wantBody {
		t.Errorf("Body = %q, want %q", parsed.Body, wantBody)
	}

	// Navigation cards never index themselves.
	if strings.Contains(parsed.Body, "[[index]]") {
		t.Errorf("draft indexes itself:\n%s", parsed.Body)
	}
}

func TestDraftIndexEmptyStore(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	draft, err := s.DraftIndex()
	if err != nil {
		t.Fatalf("DraftIndex failed: %v", err)
	}
	if _, err := UnmarshalCard("index", []byte(draft)); err != nil {
		t.Errorf("empty draft not parseable: %v\n%s", err, draft)
	}
}
