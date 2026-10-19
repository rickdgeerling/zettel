package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RecentChange pairs a recently modified card with its link partners,
// split by direction: Inbound are cards linking to it, Outbound are cards
// it links to. The organize skill uses these pairs to decide between
// "genuine contradiction" and "evolution".
type RecentChange struct {
	Slug     string   `json:"slug"`
	Modified string   `json:"modified"`
	Inbound  []string `json:"inbound"`  // ← cards referencing this card
	Outbound []string `json:"outbound"` // → cards this card references
}

// OrganizeReport is the output of a maintenance pass: the set of orphans,
// hubs, unresolved conflicts, recently modified cards (with their neighbors),
// and broken links.
type OrganizeReport struct {
	Total            int            `json:"total"`
	Orphans          []string       `json:"orphans"`
	Hubs             []string       `json:"hubs"`
	Conflicts        []string       `json:"conflicts"`
	RecentlyModified []RecentChange `json:"recently_modified"`
	BrokenLinks      []BrokenLink   `json:"broken_links"`
}

// OrganizeReport builds the maintenance report for active cards. When since
// is non-zero, RecentlyModified is limited to cards modified on or after that
// date (orphans, hubs, and conflicts are always computed across all cards).
// Navigation cards (the keyword index and category sub-indexes) are never
// reported as orphans: they link outward by design.
func (s *Store) OrganizeReport(since time.Time) (*OrganizeReport, error) {
	graph, err := s.LinkGraph()
	if err != nil {
		return nil, err
	}

	report := &OrganizeReport{
		Total:            len(graph),
		Orphans:          []string{},
		Hubs:             []string{},
		Conflicts:        []string{},
		RecentlyModified: []RecentChange{},
		BrokenLinks:      []BrokenLink{},
	}

	entries, err := os.ReadDir(s.cardsDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		slug := strings.TrimSuffix(entry.Name(), ".md")

		data, err := os.ReadFile(filepath.Join(s.cardsDir, entry.Name()))
		if err != nil {
			continue
		}
		card, err := UnmarshalCard(slug, data)
		if err != nil {
			continue
		}

		stats := graph[slug]
		if stats.Inbound == 0 && !isIndexSlug(slug) {
			report.Orphans = append(report.Orphans, slug)
		} else if stats.Inbound >= 10 {
			report.Hubs = append(report.Hubs, slug)
		}

		if card.Status == "conflict" {
			report.Conflicts = append(report.Conflicts, slug)
		}

		if since.IsZero() || !card.Modified.Before(since) {
			inbound, outbound := s.linkedNeighbors(slug)
			report.RecentlyModified = append(report.RecentlyModified, RecentChange{
				Slug:     slug,
				Modified: card.Modified.Format("2006-01-02"),
				Inbound:  inbound,
				Outbound: outbound,
			})
		}
	}

	sort.Strings(report.Orphans)
	sort.Strings(report.Hubs)
	sort.Strings(report.Conflicts)
	sort.Slice(report.RecentlyModified, func(i, j int) bool {
		return report.RecentlyModified[i].Slug < report.RecentlyModified[j].Slug
	})

	if broken, err := s.BrokenLinks(); err == nil && broken != nil {
		report.BrokenLinks = broken
	}

	return report, nil
}

// isIndexSlug reports whether slug is a navigation card: the keyword
// index or a category sub-index. Index cards exist to link outward and
// are never legitimately "orphaned", so organize reports skip them.
func isIndexSlug(slug string) bool {
	return slug == "index" || strings.HasSuffix(slug, "-index")
}

// linkedNeighbors returns the sorted inbound and outbound link partners
// for a card so report consumers can tell direction.
func (s *Store) linkedNeighbors(slug string) (inbound []string, outbound []string) {
	inbound, outbound, err := s.LinksForCard(slug)
	if err != nil {
		return []string{}, []string{}
	}
	if inbound == nil {
		inbound = []string{}
	}
	if outbound == nil {
		outbound = []string{}
	}
	sort.Strings(inbound)
	sort.Strings(outbound)
	return inbound, outbound
}

// StampLastOrganize records the date of this organize run so the next
// incremental run can use it as --since.
func (s *Store) StampLastOrganize() error {
	return s.StampLastOrganizeAt(time.Now())
}

// StampLastOrganizeAt is StampLastOrganize with an explicit timestamp
// (UTC, truncated to the day) for testability.
func (s *Store) StampLastOrganizeAt(t time.Time) error {
	path := filepath.Join(s.Root, LastOrganizeFile)
	content := []byte(t.UTC().Format("2006-01-02") + "\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
