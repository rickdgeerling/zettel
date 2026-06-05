package store

import (
	"os"
	"path/filepath"
	"strings"
)

type CardMetadata struct {
	Slug     string
	Title    string
	Tags     []string
	Category string
	Status   string
	Created  string
	Modified string
}

func (s *Store) SearchCards(query string, tags []string, category, status *string, limit, offset int) ([]CardMetadata, error) {
	entries, err := os.ReadDir(s.cardsDir)
	if err != nil {
		return nil, err
	}

	var results []CardMetadata
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

		if !matchesQuery(card, query, tags, category, status) {
			continue
		}

		results = append(results, CardMetadata{
			Slug:     card.Slug,
			Title:    card.Title,
			Tags:     card.Tags,
			Category: card.Category,
			Status:   card.Status,
			Created:  card.Created.Format("2006-01-02"),
			Modified: card.Modified.Format("2006-01-02"),
		})
	}

	// Apply pagination
	if offset > 0 {
		if offset >= len(results) {
			return nil, nil
		}
		results = results[offset:]
	}
	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}

	return results, nil
}

func matchesQuery(card *Card, query string, tags []string, category, status *string) bool {
	if query != "" {
		q := strings.ToLower(query)
		found := false
		if strings.Contains(strings.ToLower(card.Title), q) {
			found = true
		}
		if strings.Contains(strings.ToLower(card.Body), q) {
			found = true
		}
		for _, tag := range card.Tags {
			if strings.Contains(strings.ToLower(tag), q) {
				found = true
				break
			}
		}
		if strings.Contains(strings.ToLower(card.Category), q) {
			found = true
		}
		if !found {
			return false
		}
	}

	for _, tag := range tags {
		found := false
		for _, ctag := range card.Tags {
			if ctag == tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if category != nil && *category != "" && card.Category != *category {
		return false
	}
	if status != nil && *status != "" && card.Status != *status {
		return false
	}

	return true
}
