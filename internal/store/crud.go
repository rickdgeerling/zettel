package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var ErrCardNotFound = fmt.Errorf("card not found")

func (s *Store) ReadCard(slug string) (*Card, error) {
	path := filepath.Join(s.cardsDir, slug+".md")
	if data, err := os.ReadFile(path); err == nil {
		card, err := UnmarshalCard(slug, data)
		if err != nil {
			return nil, fmt.Errorf("parsing card %s: %w", slug, err)
		}
		return card, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading card %s: %w", slug, err)
	}

	path = filepath.Join(s.archDir, slug+".md")
	if data, err := os.ReadFile(path); err == nil {
		card, err := UnmarshalCard(slug, data)
		if err != nil {
			return nil, fmt.Errorf("parsing archived card %s: %w", slug, err)
		}
		return card, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading archived card %s: %w", slug, err)
	}

	return nil, ErrCardNotFound
}

func (s *Store) WriteCard(slug string, card *Card, source string) error {
	if err := validateSlug(slug); err != nil {
		return fmt.Errorf("invalid slug: %w", err)
	}
	if card.Title == "" {
		return fmt.Errorf("title is required")
	}
	if card.Created.IsZero() {
		return fmt.Errorf("created is required")
	}

	if card.Modified.IsZero() {
		card.Modified = time.Now().UTC()
	}
	if card.Source == "" {
		card.Source = source
	}

	data := card.Marshal()
	path := filepath.Join(s.cardsDir, slug+".md")
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		return fmt.Errorf("writing card: %w", err)
	}

	return nil
}

func (s *Store) ArchiveCard(slug string) error {
	srcPath := filepath.Join(s.cardsDir, slug+".md")
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("card %q not found in active cards: %w", slug, ErrCardNotFound)
	}

	dstPath := filepath.Join(s.archDir, slug+".md")
	if _, err := os.Stat(dstPath); err == nil {
		return fmt.Errorf("card %q already archived", slug)
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("reading card for archive: %w", err)
	}
	if err := os.WriteFile(dstPath, data, 0644); err != nil {
		return fmt.Errorf("writing to archived: %w", err)
	}
	if err := os.Remove(srcPath); err != nil {
		return fmt.Errorf("removing original: %w", err)
	}

	return nil
}
