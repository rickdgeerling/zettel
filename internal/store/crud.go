package store

import (
	"errors"
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

	// Immutable metadata comes from the card already on disk (active or
	// archived): created is preserved so callers carrying stale copies
	// cannot reset it, and modified is always bumped so --modified-since
	// windows observe every rewrite.
	existing, err := s.ReadCard(slug)
	switch {
	case err == nil:
		if !existing.Created.IsZero() {
			card.Created = existing.Created
		}
		card.Modified = time.Now().Truncate(time.Second)
	case errors.Is(err, ErrCardNotFound):
		// New card: trust caller-provided timestamps (imports may carry
		// historical ones), defaulting to now.
		if card.Created.IsZero() {
			card.Created = time.Now().Truncate(time.Second)
		}
		if card.Modified.IsZero() {
			card.Modified = time.Now().Truncate(time.Second)
		}
	default:
		return fmt.Errorf("reading existing card %s: %w", slug, err)
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
