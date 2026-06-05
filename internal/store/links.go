package store

import (
	"os"
	"path/filepath"
)

type LinkStats struct {
	Inbound  int
	Outbound int
}

func (s *Store) LinkGraph() (map[string]LinkStats, error) {
	graph := make(map[string]LinkStats)

	if err := s.scanLinks(s.cardsDir, graph); err != nil {
		return nil, err
	}

	if err := s.scanLinks(s.archDir, graph); err != nil {
		return nil, err
	}

	return graph, nil
}

func (s *Store) scanLinks(dir string, graph map[string]LinkStats) error {
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

		cardSlug := entry.Name()[:len(entry.Name())-3]
		links := ExtractWikilinks(string(data))

		if _, exists := graph[cardSlug]; !exists {
			graph[cardSlug] = LinkStats{}
		}

		stats := graph[cardSlug]
		stats.Outbound = len(links)
		graph[cardSlug] = stats

		for _, target := range links {
			if _, exists := graph[target]; !exists {
				graph[target] = LinkStats{}
			}
			tgtStats := graph[target]
			tgtStats.Inbound++
			graph[target] = tgtStats
		}
	}

	return nil
}

func (s *Store) LinksForCard(slug string) (inbound []string, outbound []string, err error) {
	_, err = s.ReadCard(slug)
	if err != nil {
		return nil, nil, err
	}

	inbound = s.scanInbound(slug, s.cardsDir)
	inbound = append(inbound, s.scanInbound(slug, s.archDir)...)

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
				result = append(result, entry.Name()[:len(entry.Name())-3])
			}
		}
	}
	return result
}
