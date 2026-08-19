package store

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type LinkStats struct {
	Inbound  int
	Outbound int
	Archived bool
}

type BrokenLink struct {
	From string // active card slug containing the link
	To   string // target slug
	Kind string // "missing" or "archived"
}

// LinkGraph returns the active link graph. Only cards in cardsDir are nodes;
// links to missing or archived slugs do not create phantom nodes. Archived
// cards and their links are excluded entirely.
func (s *Store) LinkGraph() (map[string]LinkStats, error) {
	return s.linkGraph(false)
}

// LinkGraphIncludingArchived is like LinkGraph but also includes archived
// cards as nodes (marked Archived) and counts their links.
func (s *Store) LinkGraphIncludingArchived() (map[string]LinkStats, error) {
	return s.linkGraph(true)
}

func (s *Store) linkGraph(includeArchived bool) (map[string]LinkStats, error) {
	active, err := s.listSlugs(s.cardsDir)
	if err != nil {
		return nil, err
	}

	archived := map[string]bool{}
	if includeArchived {
		archived, err = s.listSlugs(s.archDir)
		if err != nil {
			return nil, err
		}
	}

	graph := make(map[string]LinkStats)

	// Register nodes first so orphan cards still appear with zero counts.
	for slug := range active {
		graph[slug] = LinkStats{}
	}
	for slug := range archived {
		if _, ok := graph[slug]; !ok {
			graph[slug] = LinkStats{Archived: true}
		}
	}

	if err := s.scanDirLinks(s.cardsDir, graph, false); err != nil {
		return nil, err
	}
	if includeArchived {
		if err := s.scanDirLinks(s.archDir, graph, true); err != nil {
			return nil, err
		}
	}

	return graph, nil
}

// scanDirLinks computes outbound counts for cards in dir and increments
// inbound counts on link targets. Targets that are not registered nodes are
// skipped, so links to missing or unregistered slugs never create phantom
// graph entries.
func (s *Store) scanDirLinks(dir string, graph map[string]LinkStats, archived bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		slug := strings.TrimSuffix(entry.Name(), ".md")
		links := ExtractWikilinks(string(data))

		stats, ok := graph[slug]
		if !ok {
			stats = LinkStats{Archived: archived}
		}
		stats.Outbound = len(links)
		graph[slug] = stats

		for _, target := range links {
			tgt, ok := graph[target]
			if !ok {
				continue
			}
			tgt.Inbound++
			graph[target] = tgt
		}
	}

	return nil
}

// BrokenLinks reports [[wikilink]] references in active cards whose target
// does not exist as an active card. Targets that are archived are marked
// "archived"; everything else is "missing".
func (s *Store) BrokenLinks() ([]BrokenLink, error) {
	active, err := s.listSlugs(s.cardsDir)
	if err != nil {
		return nil, err
	}
	archived, err := s.listSlugs(s.archDir)
	if err != nil {
		return nil, err
	}

	var broken []BrokenLink
	entries, err := os.ReadDir(s.cardsDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.cardsDir, entry.Name()))
		if err != nil {
			continue
		}

		from := strings.TrimSuffix(entry.Name(), ".md")
		for _, target := range ExtractWikilinks(string(data)) {
			if active[target] {
				continue
			}
			kind := "missing"
			if archived[target] {
				kind = "archived"
			}
			broken = append(broken, BrokenLink{From: from, To: target, Kind: kind})
		}
	}

	sort.Slice(broken, func(i, j int) bool {
		if broken[i].From != broken[j].From {
			return broken[i].From < broken[j].From
		}
		return broken[i].To < broken[j].To
	})

	return broken, nil
}

// listSlugs returns the set of card slugs (filenames without .md) in a
// directory.
func (s *Store) listSlugs(dir string) (map[string]bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	slugs := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		slugs[strings.TrimSuffix(entry.Name(), ".md")] = true
	}
	return slugs, nil
}

func (s *Store) LinksForCard(slug string) (inbound []string, outbound []string, err error) {
	_, err = s.ReadCard(slug)
	if err != nil {
		return nil, nil, err
	}

	// Inbound links come from active cards only. Archived cards are out of
	// the active network and should not contribute to a card's inbound count.
	inbound = s.scanInbound(slug, s.cardsDir)

	card, err := s.ReadCard(slug)
	if err != nil {
		return inbound, nil, err
	}
	outbound = ExtractWikilinks(card.Body)

	return inbound, outbound, nil
}

func (s *Store) scanInbound(slug, dir string) []string {
	var result []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return result
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		links := ExtractWikilinks(string(data))
		for _, link := range links {
			if link == slug {
				result = append(result, strings.TrimSuffix(entry.Name(), ".md"))
			}
		}
	}
	return result
}
