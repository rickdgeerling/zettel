package store

import (
	"bytes"
	"fmt"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

type Card struct {
	Slug     string
	Title    string    `yaml:"title"`
	Created  time.Time `yaml:"created"`
	Modified time.Time `yaml:"modified"`
	Source   string    `yaml:"source"`
	Tags     []string  `yaml:"tags"`
	Category string    `yaml:"category"`
	Status   string    `yaml:"status"`
	// Description is an optional one-line summary of the card. The
	// keyword index renders it in place of the title, and search matches
	// it, so recall can surface what a card is about without reading it.
	Description string
	Body        string
}

type frontmatter struct {
	Title       string    `yaml:"title"`
	Created     time.Time `yaml:"created"`
	Modified    time.Time `yaml:"modified"`
	Source      string    `yaml:"source"`
	Tags        []string  `yaml:"tags"`
	Category    string    `yaml:"category"`
	Status      string    `yaml:"status"`
	Description string    `yaml:"description,omitempty"`
}

var wikilinkRegex = regexp.MustCompile(`\[\[([a-z0-9][a-z0-9-]*[a-z0-9])\]\]`)

func validateSlug(slug string) error {
	if len(slug) < 2 || len(slug) > 60 {
		return fmt.Errorf("slug must be 2-60 characters")
	}
	if slug[0] == '-' || slug[len(slug)-1] == '-' {
		return fmt.Errorf("slug cannot start or end with hyphen")
	}
	for _, c := range slug {
		if c >= 'A' && c <= 'Z' {
			return fmt.Errorf("slug must be lowercase")
		}
		if c == '_' {
			return fmt.Errorf("slug must be kebab-case alphanumeric")
		}
		if c != '-' && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return fmt.Errorf("slug must be kebab-case alphanumeric")
		}
	}
	return nil
}

func ExtractWikilinks(body string) []string {
	matches := wikilinkRegex.FindAllStringSubmatch(body, -1)
	result := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	for _, m := range matches {
		link := m[1]
		if !seen[link] {
			result = append(result, link)
			seen[link] = true
		}
	}
	return result
}

func (c *Card) Marshal() string {
	fm := frontmatter{
		Title:       c.Title,
		Created:     c.Created,
		Modified:    c.Modified,
		Source:      c.Source,
		Tags:        c.Tags,
		Category:    c.Category,
		Status:      c.Status,
		Description: c.Description,
	}

	var buf bytes.Buffer
	// Standard markdown frontmatter delimiters. Read output is a valid
	// write input, so agents can pipe `zettel read s | zettel write s`
	// without reconstructing anything — and the on-disk format matches
	// what every markdown tool expects.
	buf.WriteString("---\n")
	data, _ := yaml.Marshal(fm)
	buf.Write(data)
	buf.WriteString("---\n")
	buf.WriteString(c.Body)
	return buf.String()
}

func UnmarshalCard(slug string, data []byte) (*Card, error) {
	parts := splitFrontmatterBody(data)
	if len(parts) < 2 {
		return nil, fmt.Errorf("card must contain frontmatter and body separated by ---")
	}

	fm := &frontmatter{}
	if err := yaml.Unmarshal([]byte(parts[0]), fm); err != nil {
		return nil, fmt.Errorf("invalid frontmatter: %w", err)
	}

	body := ""
	if len(parts) >= 2 {
		body = parts[1]
	}

	return &Card{
		Slug:        slug,
		Title:       fm.Title,
		Created:     fm.Created,
		Modified:    fm.Modified,
		Source:      fm.Source,
		Tags:        fm.Tags,
		Category:    fm.Category,
		Status:      fm.Status,
		Description: fm.Description,
		Body:        body,
	}, nil
}

func splitFrontmatterBody(data []byte) []string {
	marker := []byte("\n---\n")
	idx := bytes.Index(data, marker)
	if idx == -1 {
		return []string{string(data), ""}
	}
	return []string{string(data[:idx]), string(data[idx+len(marker):])}
}
