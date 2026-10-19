package store

import (
	"testing"
	"time"
)

func TestLinkGraph(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	cards := []*Card{
		{Slug: "card-a", Title: "Card A", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Body: "Links to [[card-b]] and [[card-c]]"},
		{Slug: "card-b", Title: "Card B", Created: time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC), Body: "Linked from [[card-a]]"},
		{Slug: "card-c", Title: "Card C", Created: time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC), Body: "Linked from [[card-a]], links to [[card-b]]"},
		{Slug: "card-d", Title: "Card D", Created: time.Date(2025, 1, 18, 0, 0, 0, 0, time.UTC), Body: "No links"},
	}
	for _, c := range cards {
		_ = s.WriteCard(c.Slug, c, "test")
	}

	graph, err := s.LinkGraph()
	if err != nil {
		t.Fatalf("LinkGraph failed: %v", err)
	}

	aStats := graph["card-a"]
	if aStats.Inbound != 2 {
		t.Errorf("card-a inbound = %d, want 2", aStats.Inbound)
	}
	if aStats.Outbound != 2 {
		t.Errorf("card-a outbound = %d, want 2", aStats.Outbound)
	}

	bStats := graph["card-b"]
	if bStats.Inbound != 2 {
		t.Errorf("card-b inbound = %d, want 2", bStats.Inbound)
	}

	cStats := graph["card-c"]
	if cStats.Inbound != 1 {
		t.Errorf("card-c inbound = %d, want 1", cStats.Inbound)
	}
	if cStats.Outbound != 2 {
		t.Errorf("card-c outbound = %d, want 2", cStats.Outbound)
	}

	dStats := graph["card-d"]
	if dStats.Inbound != 0 {
		t.Errorf("card-d inbound = %d, want 0", dStats.Inbound)
	}
	if dStats.Outbound != 0 {
		t.Errorf("card-d outbound = %d, want 0", dStats.Outbound)
	}
}

func TestLinkGraphExcludesArchivedByDefault(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	cardA := &Card{Slug: "card-a", Title: "Card A", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Body: "Links to [[card-b]]"}
	_ = s.WriteCard("card-a", cardA, "test")

	cardB := &Card{Slug: "card-b", Title: "Card B", Created: time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC), Body: "Content"}
	_ = s.WriteCard("card-b", cardB, "test")
	_ = s.ArchiveCard("card-b")

	graph, _ := s.LinkGraph()

	if _, exists := graph["card-b"]; exists {
		t.Error("card-b should not appear in the default active-only graph")
	}

	aStats := graph["card-a"]
	if aStats.Outbound != 1 {
		t.Errorf("card-a outbound = %d, want 1", aStats.Outbound)
	}
	if aStats.Inbound != 0 {
		t.Errorf("card-a inbound = %d, want 0", aStats.Inbound)
	}
}

func TestLinkGraphIncludingArchived(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	cardA := &Card{Slug: "card-a", Title: "Card A", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Body: "Links to [[card-b]]"}
	_ = s.WriteCard("card-a", cardA, "test")

	cardB := &Card{Slug: "card-b", Title: "Card B", Created: time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC), Body: "Content"}
	_ = s.WriteCard("card-b", cardB, "test")
	_ = s.ArchiveCard("card-b")

	graph, _ := s.LinkGraphIncludingArchived()

	bStats, exists := graph["card-b"]
	if !exists {
		t.Fatal("card-b not in graph with archived included")
	}
	if !bStats.Archived {
		t.Error("card-b should be marked archived")
	}
	if bStats.Inbound != 1 {
		t.Errorf("card-b inbound = %d, want 1", bStats.Inbound)
	}
}

func TestLinkGraphEmpty(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	graph, err := s.LinkGraph()
	if err != nil {
		t.Fatalf("LinkGraph failed: %v", err)
	}
	if len(graph) != 0 {
		t.Errorf("Expected empty graph, got %d entries", len(graph))
	}
}

func TestLinksForCard(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	cards := []*Card{
		{Slug: "card-a", Title: "Card A", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Body: "Links to [[card-b]]"},
		{Slug: "card-b", Title: "Card B", Created: time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC), Body: "Referenced by [[card-a]]"},
		{Slug: "card-c", Title: "Card C", Created: time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC), Body: "Links to [[card-a]]"},
	}
	for _, c := range cards {
		_ = s.WriteCard(c.Slug, c, "test")
	}

	inbound, outbound, err := s.LinksForCard("card-a")
	if err != nil {
		t.Fatalf("LinksForCard failed: %v", err)
	}

	if len(inbound) != 2 || inbound[0] != "card-b" || inbound[1] != "card-c" {
		t.Errorf("Inbound = %v, want [card-b card-c]", inbound)
	}
	if len(outbound) != 1 || outbound[0] != "card-b" {
		t.Errorf("Outbound = %v, want [card-b]", outbound)
	}
}

func TestLinksForCardNotFound(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	_, _, err := s.LinksForCard("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent card")
	}
}

func TestLinksForCardArchived(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	cardA := &Card{Slug: "card-a", Title: "Card A", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Body: "Links to [[card-b]]"}
	_ = s.WriteCard("card-a", cardA, "test")

	cardB := &Card{Slug: "card-b", Title: "Card B", Created: time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC), Body: "Content"}
	_ = s.WriteCard("card-b", cardB, "test")
	_ = s.ArchiveCard("card-b")

	inbound, outbound, err := s.LinksForCard("card-b")
	if err != nil {
		t.Fatalf("LinksForCard failed for archived card: %v", err)
	}
	if len(inbound) != 1 || inbound[0] != "card-a" {
		t.Errorf("Inbound = %v, want [card-a]", inbound)
	}
	if len(outbound) != 0 {
		t.Errorf("Outbound = %v, want []", outbound)
	}
}

func TestBrokenLinks(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	cards := []*Card{
		{Slug: "card-a", Title: "Card A", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Body: "Links to [[missing-one]] and [[archived-one]] and [[card-b]]"},
		{Slug: "card-b", Title: "Card B", Created: time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC), Body: "Content"},
	}
	for _, c := range cards {
		_ = s.WriteCard(c.Slug, c, "test")
	}

	// Create and archive a target so we can verify the "archived" kind.
	archived := &Card{Slug: "archived-one", Title: "Archived One", Created: time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC), Body: "Content"}
	_ = s.WriteCard("archived-one", archived, "test")
	_ = s.ArchiveCard("archived-one")

	broken, err := s.BrokenLinks()
	if err != nil {
		t.Fatalf("BrokenLinks failed: %v", err)
	}

	if len(broken) != 2 {
		t.Fatalf("Expected 2 broken links, got %d: %v", len(broken), broken)
	}

	byTarget := map[string]BrokenLink{}
	for _, b := range broken {
		byTarget[b.To] = b
	}

	if b, ok := byTarget["missing-one"]; !ok || b.Kind != "missing" {
		t.Errorf("missing-one link missing or wrong kind: %+v", b)
	}
	if b, ok := byTarget["archived-one"]; !ok || b.Kind != "archived" {
		t.Errorf("archived-one link missing or wrong kind: %+v", b)
	}
	if b := byTarget["archived-one"]; b.From != "card-a" {
		t.Errorf("archived-one link From = %q, want card-a", b.From)
	}
}

func TestBrokenLinksNone(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	cards := []*Card{
		{Slug: "card-a", Title: "Card A", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Body: "Links to [[card-b]]"},
		{Slug: "card-b", Title: "Card B", Created: time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC), Body: "Content"},
	}
	for _, c := range cards {
		_ = s.WriteCard(c.Slug, c, "test")
	}

	broken, err := s.BrokenLinks()
	if err != nil {
		t.Fatalf("BrokenLinks failed: %v", err)
	}
	if len(broken) != 0 {
		t.Errorf("Expected no broken links, got %v", broken)
	}
}

func TestBrokenLinkSuggestions(t *testing.T) {
	tmp := t.TempDir()
	s, _ := Init(tmp)

	// Near-miss targets taken from a real organize run: each broken slug
	// is a wrong name for an existing concept.
	cards := []*Card{
		{Slug: "linker", Title: "Linker", Created: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Body: "Refs [[notification-opt-in-vs-out]], [[observability-agent-memory]], [[deterministic-job-id-dedup]] and [[zzz-qqq]]"},
		{Slug: "incident-notification-opt-in-vs-out", Title: "Incident Notifications", Created: time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC), Body: "Content"},
		{Slug: "agent-memory-edit-tool-design", Title: "Agent Memory Edits", Created: time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC), Body: "Content"},
		{Slug: "bull-job-id-concurrency-filtering", Title: "Bull Job Ids", Created: time.Date(2025, 1, 18, 0, 0, 0, 0, time.UTC), Body: "Content"},
		{Slug: "go-gc-pauses", Title: "GC Pauses", Created: time.Date(2025, 1, 19, 0, 0, 0, 0, time.UTC), Body: "Content"},
	}
	for _, c := range cards {
		_ = s.WriteCard(c.Slug, c, "test")
	}

	broken, err := s.BrokenLinks()
	if err != nil {
		t.Fatalf("BrokenLinks failed: %v", err)
	}

	byTarget := map[string]BrokenLink{}
	for _, b := range broken {
		byTarget[b.To] = b
	}

	wantSuggestion := map[string]string{
		"notification-opt-in-vs-out": "incident-notification-opt-in-vs-out", // substring embed
		"observability-agent-memory": "agent-memory-edit-tool-design",       // shared "agent-memory" run
		"deterministic-job-id-dedup": "bull-job-id-concurrency-filtering",   // shared "job-id" run
	}
	for target, want := range wantSuggestion {
		got := byTarget[target].Suggestion
		if got != want {
			t.Errorf("suggestion for %q = %q, want %q", target, got, want)
		}
	}

	if got := byTarget["zzz-qqq"].Suggestion; got != "" {
		t.Errorf("dissimilar target %q got suggestion %q, want none", "zzz-qqq", got)
	}
}
