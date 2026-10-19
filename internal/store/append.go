package store

import "strings"

// AppendToSection appends a single line of text to the card's
// `## <section>` section, creating the section at the end of the body
// when absent. Existing prose is never modified — the line is inserted
// at the end of the section's content — making append-only edits (e.g.
// the organize skill's `## Links` updates) safe by construction. The
// card's modified timestamp is bumped.
func (s *Store) AppendToSection(slug, section, line string) error {
	card, err := s.ReadCard(slug)
	if err != nil {
		return err
	}

	card.Body = appendLineToSection(card.Body, section, line)

	return s.WriteCard(slug, card, "append")
}

// appendLineToSection inserts line at the end of the `## <section>`
// block in body, appending a new section when the heading is absent.
// The section argument accepts "Links" or "## Links"; surrounding
// whitespace and heading hashes are normalized away.
func appendLineToSection(body, section, line string) string {
	section = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(section), "#"))
	heading := "## " + section
	lines := strings.Split(body, "\n")

	idx := -1
	for i, l := range lines {
		if strings.TrimRight(l, " \t") == heading {
			idx = i
			break
		}
	}

	if idx == -1 {
		// Section absent: append it at the end of the body.
		trimmed := strings.TrimRight(body, "\n")
		if trimmed == "" {
			return heading + "\n\n" + line + "\n"
		}
		return trimmed + "\n\n" + heading + "\n\n" + line + "\n"
	}

	// Section exists: insert after its last non-empty line, before the
	// next heading (or end of body).
	end := len(lines)
	for i := idx + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimLeft(lines[i], " \t"), "#") {
			end = i
			break
		}
	}
	last := idx
	for i := end - 1; i > idx; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			last = i
			break
		}
	}

	var out []string
	if last == idx {
		// Empty section: blank line after the heading, then the new line.
		out = append(out, lines[:idx+1]...)
		out = append(out, "", line)
		out = append(out, lines[idx+1:]...)
	} else {
		out = append(out, lines[:last+1]...)
		out = append(out, line)
		out = append(out, lines[last+1:]...)
	}

	result := strings.Join(out, "\n")
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}
