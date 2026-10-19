package store

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type CardMetadata struct {
	Slug        string
	Title       string
	Tags        []string
	Category    string
	Status      string
	Description string
	Created     string
	Modified    string
}

// SearchOptions describes a card search. All fields are optional; the zero
// value returns every active card. ModifiedSince filters to cards whose
// frontmatter `modified` timestamp is on or after the given time.
type SearchOptions struct {
	Query         string
	Tags          []string
	Category      string
	Status        string
	ModifiedSince time.Time
	Limit         int
	Offset        int
}

// Search returns active cards matching the given options, sorted by slug and
// paginated by Limit/Offset.
func (s *Store) Search(opts SearchOptions) ([]CardMetadata, error) {
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

		if !matchesSearch(card, opts) {
			continue
		}

		results = append(results, CardMetadata{
			Slug:        card.Slug,
			Title:       card.Title,
			Tags:        card.Tags,
			Category:    card.Category,
			Status:      card.Status,
			Description: card.Description,
			Created:     card.Created.Format("2006-01-02"),
			Modified:    card.Modified.Format("2006-01-02"),
		})
	}

	// Deterministic ordering so pagination is stable across runs.
	sort.Slice(results, func(i, j int) bool { return results[i].Slug < results[j].Slug })

	// Apply pagination.
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(results) {
		return nil, nil
	}
	results = results[offset:]

	if opts.Limit > 0 && opts.Limit < len(results) {
		results = results[:opts.Limit]
	}

	return results, nil
}

// SearchCards is retained for backward compatibility. New callers should use
// Search with SearchOptions.
func (s *Store) SearchCards(query string, tags []string, category, status *string, limit, offset int) ([]CardMetadata, error) {
	opts := SearchOptions{
		Query:  query,
		Tags:   tags,
		Limit:  limit,
		Offset: offset,
	}
	if category != nil {
		opts.Category = *category
	}
	if status != nil {
		opts.Status = *status
	}
	return s.Search(opts)
}

func matchesSearch(card *Card, opts SearchOptions) bool {
	if opts.Query != "" {
		q := strings.ToLower(opts.Query)
		found := false
		if strings.Contains(strings.ToLower(card.Title), q) {
			found = true
		}
		if strings.Contains(strings.ToLower(card.Description), q) {
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

	for _, tag := range opts.Tags {
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

	if opts.Category != "" && card.Category != opts.Category {
		return false
	}
	if opts.Status != "" && card.Status != opts.Status {
		return false
	}
	if !opts.ModifiedSince.IsZero() && card.Modified.Before(opts.ModifiedSince) {
		return false
	}

	return true
}
