package store

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RecentChange pairs a recently modified card with the neighbors it links to
// or is linked from. The organize skill uses these pairs to decide between
// "genuine contradiction" and "evolution".
type RecentChange struct {
	Slug      string   `json:"slug"`
	Modified  string   `json:"modified"`
	Neighbors []string `json:"neighbors"`
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
		if stats.Inbound == 0 {
			report.Orphans = append(report.Orphans, slug)
		} else if stats.Inbound >= 10 {
			report.Hubs = append(report.Hubs, slug)
		}

		if card.Status == "conflict" {
			report.Conflicts = append(report.Conflicts, slug)
		}

		if since.IsZero() || !card.Modified.Before(since) {
			report.RecentlyModified = append(report.RecentlyModified, RecentChange{
				Slug:      slug,
				Modified:  card.Modified.Format("2006-01-02"),
				Neighbors: s.linkedNeighbors(slug),
			})
		}
	}

	sort.Strings(report.Orphans)
	sort.Strings(report.Hubs)
	sort.Strings(report.Conflicts)
	sort.Slice(report.RecentlyModified, func(i, j int) bool {
		return report.RecentlyModified[i].Slug < report.RecentlyModified[j].Slug
	})

	if broken, err := s.BrokenLinks(); err == nil {
		report.BrokenLinks = broken
	}

	return report, nil
}

func (s *Store) linkedNeighbors(slug string) []string {
	inbound, outbound, err := s.LinksForCard(slug)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var neighbors []string
	for _, n := range append(inbound, outbound...) {
		if !seen[n] {
			seen[n] = true
			neighbors = append(neighbors, n)
		}
	}
	sort.Strings(neighbors)
	return neighbors
}
