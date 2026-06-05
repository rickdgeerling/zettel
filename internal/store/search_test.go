package store

import (
	"testing"
	"time"
)

func TestSearchCardsSubstringMatch(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	cards := []*Card{
		{Slug: "jwt-revocation", Title: "JWT Revocation Pattern", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Body: "Server-side token blacklist"},
		{Slug: "oauth2-pkce", Title: "OAuth2 PKCE Flow", Created: time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC), Body: "PKCE for public clients"},
		{Slug: "docker-network", Title: "Docker Network Config", Created: time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC), Body: "Bridge networking setup"},
	}
	for _, c := range cards {
		_ = s.WriteCard(c.Slug, c, "test")
	}

	results, err := s.SearchCards("jwt", nil, nil, nil)
	if err != nil {
		t.Fatalf("SearchCards failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'jwt', got %d", len(results))
	}
	if results[0].Slug != "jwt-revocation" {
		t.Errorf("Wrong slug: got %q", results[0].Slug)
	}

	results, err = s.SearchCards("docker", nil, nil, nil)
	if err != nil {
		t.Fatalf("SearchCards failed: %v", err)
	}
	if len(results) != 1 || results[0].Slug != "docker-network" {
		t.Errorf("Wrong results for 'docker': %v", results)
	}

	results, err = s.SearchCards("token", nil, nil, nil)
	if err != nil {
		t.Fatalf("SearchCards failed: %v", err)
	}
	if len(results) != 1 || results[0].Slug != "jwt-revocation" {
		t.Errorf("Wrong results for 'token': %v", results)
	}
}

func TestSearchCardsNoQueryReturnsEmpty(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{Slug: "some-card", Title: "Test", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Body: "Content"}
	_ = s.WriteCard("some-card", card, "test")

	results, err := s.SearchCards("", nil, nil, nil)
	if err != nil {
		t.Fatalf("SearchCards failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected empty results for no query, got %d", len(results))
	}
}

func TestSearchCardsTagFilter(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	cards := []*Card{
		{Slug: "card-a", Title: "Card A", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Tags: []string{"security", "pattern"}, Body: "Content"},
		{Slug: "card-b", Title: "Card B", Created: time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC), Tags: []string{"docker"}, Body: "Content"},
	}
	for _, c := range cards {
		_ = s.WriteCard(c.Slug, c, "test")
	}

	results, err := s.SearchCards("", []string{"security"}, nil, nil)
	if err != nil {
		t.Fatalf("SearchCards failed: %v", err)
	}
	if len(results) != 1 || results[0].Slug != "card-a" {
		t.Errorf("Expected only card-a for tag=security, got: %v", results)
	}
}

func TestSearchCardsCategoryFilter(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	cards := []*Card{
		{Slug: "card-x", Title: "Card X", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Category: "backend", Body: "Content"},
		{Slug: "card-y", Title: "Card Y", Created: time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC), Category: "frontend", Body: "Content"},
	}
	for _, c := range cards {
		_ = s.WriteCard(c.Slug, c, "test")
	}

	results, err := s.SearchCards("", nil, ptrString("backend"), nil)
	if err != nil {
		t.Fatalf("SearchCards failed: %v", err)
	}
	if len(results) != 1 || results[0].Slug != "card-x" {
		t.Errorf("Expected only card-x for category=backend, got: %v", results)
	}
}

func TestSearchCardsStatusFilter(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	cards := []*Card{
		{Slug: "card-1", Title: "Card 1", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Status: "conflict", Body: "Content"},
		{Slug: "card-2", Title: "Card 2", Created: time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC), Status: "", Body: "Content"},
	}
	for _, c := range cards {
		_ = s.WriteCard(c.Slug, c, "test")
	}

	results, err := s.SearchCards("", nil, nil, ptrString("conflict"))
	if err != nil {
		t.Fatalf("SearchCards failed: %v", err)
	}
	if len(results) != 1 || results[0].Slug != "card-1" {
		t.Errorf("Expected only card-1 for status=conflict, got: %v", results)
	}
}

func TestSearchCardsAndLogic(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	cards := []*Card{
		{Slug: "card-a", Title: "Card A", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Tags: []string{"security"}, Category: "backend", Body: "jwt content"},
		{Slug: "card-b", Title: "Card B", Created: time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC), Tags: []string{"security"}, Category: "frontend", Body: "content"},
		{Slug: "card-c", Title: "Card C", Created: time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC), Tags: []string{"docker"}, Category: "backend", Body: "content"},
	}
	for _, c := range cards {
		_ = s.WriteCard(c.Slug, c, "test")
	}

	results, err := s.SearchCards("content", []string{"security"}, ptrString("backend"), nil)
	if err != nil {
		t.Fatalf("SearchCards failed: %v", err)
	}
	if len(results) != 1 || results[0].Slug != "card-a" {
		t.Errorf("Expected only card-a, got: %v", results)
	}
}

func TestSearchCardsArchivedNotIncluded(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	card := &Card{Slug: "archived-search", Title: "Archived Card", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Body: "Content"}
	_ = s.WriteCard("archived-search", card, "test")
	_ = s.ArchiveCard("archived-search")

	results, err := s.SearchCards("archived", nil, nil, nil)
	if err != nil {
		t.Fatalf("SearchCards failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected no results for archived card, got %d", len(results))
	}
}

func ptrString(s string) *string {
	return &s
}
