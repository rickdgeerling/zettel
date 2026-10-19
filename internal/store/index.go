package store

import (
	"sort"
	"strings"
	"time"
)

// DraftIndex builds a keyword-index card draft: every active card grouped
// by its frontmatter category, with titles as placeholder descriptions.
// The draft is a valid write input — an agent edits the descriptions and
// category placement, then saves it with WriteCard("index", ...). An
// existing index card's creation date is preserved.
func (s *Store) DraftIndex() (string, error) {
	cards, err := s.Search(SearchOptions{})
	if err != nil {
		return "", err
	}

	created := time.Now().UTC().Truncate(time.Second)
	if existing, err := s.ReadCard("index"); err == nil && !existing.Created.IsZero() {
		created = existing.Created
	}

	groups := make(map[string][]CardMetadata)
	for _, c := range cards {
		if isIndexSlug(c.Slug) {
			continue // navigation cards are not indexed
		}
		cat := c.Category
		if cat == "" {
			cat = "Uncategorized"
		}
		groups[cat] = append(groups[cat], c)
	}

	cats := make([]string, 0, len(groups))
	for cat := range groups {
		cats = append(cats, cat)
	}
	sort.Strings(cats)

	var body strings.Builder
	for _, cat := range cats {
		body.WriteString("## " + cat + "\n\n")
		// Search results are sorted by slug, so entries within a
		// category come out deterministically.
		for _, c := range groups[cat] {
			desc := strings.TrimSpace(c.Description)
			desc = strings.ReplaceAll(desc, "\n", " ") // index entries are one line
			if desc == "" {
				desc = c.Title
			}
			body.WriteString("- [[" + c.Slug + "]] — " + desc + "\n")
		}
		body.WriteString("\n")
	}

	card := &Card{
		Slug:    "index",
		Title:   "Keyword Index",
		Created: created,
		Source:  "organize",
		Body:    body.String(),
	}
	return card.Marshal(), nil
}
