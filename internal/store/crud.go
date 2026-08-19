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
		card.Created = time.Now().Truncate(time.Second)
	}
	if card.Modified.IsZero() {
		card.Modified = time.Now().Truncate(time.Second)
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

// ArchiveCard moves a card from active to archived. It is idempotent:
// archiving an already-archived card is a successful no-op. A single call
// either moves the file (via an atomic rename) or returns an error.
func (s *Store) ArchiveCard(slug string) error {
	if err := validateSlug(slug); err != nil {
		return fmt.Errorf("invalid slug: %w", err)
	}

	srcPath := filepath.Join(s.cardsDir, slug+".md")
	dstPath := filepath.Join(s.archDir, slug+".md")

	srcExists := pathExists(srcPath)
	dstExists := pathExists(dstPath)

	switch {
	case srcExists:
		// Active copy wins. os.Rename atomically replaces any stale archive.
		if err := os.Rename(srcPath, dstPath); err != nil {
			return fmt.Errorf("archiving card %q: %w", slug, err)
		}
		return nil
	case dstExists:
		// Already archived — idempotent no-op.
		return nil
	default:
		return fmt.Errorf("card %q not found in active cards: %w", slug, ErrCardNotFound)
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
