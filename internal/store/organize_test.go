package store

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestOrganizeReport(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	// A hub (linked by many), an orphan, and a conflict card.
	hub := &Card{Slug: "hub", Title: "Hub", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Body: "Hub content"}
	_ = s.WriteCard("hub", hub, "test")

	orphan := &Card{Slug: "orphan", Title: "Orphan", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Body: "Orphan content"}
	_ = s.WriteCard("orphan", orphan, "test")

	conflict := &Card{Slug: "conflict", Title: "Conflict", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Status: "conflict", Body: "Conflict content"}
	_ = s.WriteCard("conflict", conflict, "test")

	for i := 0; i < 10; i++ {
		c := &Card{
			Slug:     slugForIndex(i),
			Title:    "Linker",
			Created:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			Modified: time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC),
			Body:     "Links to [[hub]]",
		}
		_ = s.WriteCard(c.Slug, c, "test")
	}

	report, err := s.OrganizeReport(time.Time{})
	if err != nil {
		t.Fatalf("OrganizeReport failed: %v", err)
	}

	if report.Total != 13 {
		t.Errorf("Total = %d, want 13", report.Total)
	}

	if !contains(report.Orphans, "orphan") {
		t.Errorf("orphan not in Orphans: %v", report.Orphans)
	}
	if !contains(report.Hubs, "hub") {
		t.Errorf("hub not in Hubs: %v", report.Hubs)
	}
	if !contains(report.Conflicts, "conflict") {
		t.Errorf("conflict not in Conflicts: %v", report.Conflicts)
	}

	// Broken link detection flows into the report.
	if len(report.BrokenLinks) != 0 {
		t.Errorf("Expected no broken links, got %v", report.BrokenLinks)
	}
}

func TestOrganizeReportSince(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	oldCard := &Card{Slug: "old", Title: "Old", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Modified: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), Body: "content"}
	_ = s.WriteCard("old", oldCard, "test")

	newCard := &Card{Slug: "new", Title: "New", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Modified: time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC), Body: "content"}
	_ = s.WriteCard("new", newCard, "test")

	report, err := s.OrganizeReport(time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("OrganizeReport failed: %v", err)
	}

	if len(report.RecentlyModified) != 1 || report.RecentlyModified[0].Slug != "new" {
		t.Errorf("RecentlyModified = %v, want only [new]", report.RecentlyModified)
	}

	// Orphans/hubs/conflicts are still computed across all cards regardless of since.
	if !contains(report.Orphans, "old") {
		t.Errorf("orphans should still include 'old' regardless of since: %v", report.Orphans)
	}
}

func TestOrganizeReportJSON(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	// A card with no links and a card with a broken link.
	plain := &Card{Slug: "plain", Title: "Plain", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Body: "no links"}
	_ = s.WriteCard("plain", plain, "test")

	linker := &Card{Slug: "linker", Title: "Linker", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Body: "Links to [[missing]]"}
	_ = s.WriteCard("linker", linker, "test")

	report, err := s.OrganizeReport(time.Time{})
	if err != nil {
		t.Fatalf("OrganizeReport failed: %v", err)
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Empty slices must marshal as [], never null — downstream tooling
	// should not have to handle both.
	for _, field := range []string{"orphans", "hubs", "conflicts", "recently_modified", "broken_links"} {
		raw, ok := doc[field]
		if !ok {
			t.Fatalf("missing field %q in JSON: %s", field, data)
		}
		if strings.TrimSpace(string(raw)) == "null" {
			t.Errorf("field %q marshals as null, want []", field)
		}
	}

	// Broken link keys must be snake-consistent lowercase, not Go field names.
	var broken []map[string]any
	if err := json.Unmarshal(doc["broken_links"], &broken); err != nil {
		t.Fatalf("Unmarshal broken_links failed: %v", err)
	}
	if len(broken) != 1 {
		t.Fatalf("broken_links = %v, want 1 entry", broken)
	}
	for _, key := range []string{"from", "to", "kind"} {
		if _, ok := broken[0][key]; !ok {
			t.Errorf("broken_links entry missing key %q: %v", key, broken[0])
		}
	}
	if _, ok := broken[0]["From"]; ok {
		t.Errorf("broken_links entry has PascalCase key %q: %v", "From", broken[0])
	}
}

func slugForIndex(i int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	return "linker-" + string(alphabet[i])
}

func contains(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}
