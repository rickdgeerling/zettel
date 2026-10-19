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
	From string `json:"from"` // active card slug containing the link
	To   string `json:"to"`   // target slug
	Kind string `json:"kind"` // "missing" or "archived"
	// Suggestion is the existing active slug most likely intended by To
	// ("did you mean"), empty when nothing is similar enough.
	Suggestion string `json:"suggestion,omitempty"`
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

	// Suggest the nearest existing slug for each broken target. Most
	// broken links are near-miss spellings of real slugs.
	candidates := make([]string, 0, len(active))
	for slug := range active {
		candidates = append(candidates, slug)
	}
	sort.Strings(candidates) // deterministic tie-breaking
	for i := range broken {
		// Suggesting the referring card itself is never actionable.
		broken[i].Suggestion = suggestSlug(broken[i].To, broken[i].From, candidates)
	}

	sort.Slice(broken, func(i, j int) bool {
		if broken[i].From != broken[j].From {
			return broken[i].From < broken[j].From
		}
		return broken[i].To < broken[j].To
	})

	return broken, nil
}

// suggestSlug returns the candidate most likely intended by a broken
// link target, or "" when no candidate is similar enough to be a
// plausible correction.
const suggestThreshold = 0.4

func suggestSlug(target, from string, candidates []string) string {
	best := ""
	bestScore := 0.0
	for _, c := range candidates {
		if c == target || c == from {
			continue
		}
		if score := slugSimilarity(target, c); score > bestScore {
			best, bestScore = c, score
		}
	}
	if bestScore >= suggestThreshold {
		return best
	}
	return ""
}

// slugSimilarity scores how alike two slugs are on a 0..1 scale. Slugs
// share vocabulary by domain, so several weak signals are combined and
// the strongest wins: token-set overlap, longest shared run of
// consecutive tokens, substring containment, and edit distance.
func slugSimilarity(a, b string) float64 {
	score := tokenOverlap(a, b)
	if s := sharedRunScore(a, b); s > score {
		score = s
	}
	if s := substringScore(a, b); s > score {
		score = s
	}
	if s := editDistanceScore(a, b); s > score {
		score = s
	}
	return score
}

// tokenOverlap is the Jaccard similarity of the hyphen-separated tokens.
func tokenOverlap(a, b string) float64 {
	ta := strings.Split(a, "-")
	tb := strings.Split(b, "-")
	set := make(map[string]bool, len(ta))
	for _, t := range ta {
		set[t] = true
	}
	inter := 0
	for _, t := range tb {
		if set[t] {
			inter++
		}
	}
	union := len(ta) + len(tb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// sharedRunScore measures the longest run of consecutive tokens shared
// by both slugs ("agent-memory" inside both slugs), scaled by the
// shorter token count. Runs shorter than two tokens carry no signal.
func sharedRunScore(a, b string) float64 {
	run := longestSharedTokenRun(a, b)
	if run < 2 {
		return 0
	}
	minTokens := len(strings.Split(a, "-"))
	if l := len(strings.Split(b, "-")); l < minTokens {
		minTokens = l
	}
	return float64(run) / float64(minTokens)
}

func longestSharedTokenRun(a, b string) int {
	ta := strings.Split(a, "-")
	tb := strings.Split(b, "-")
	best := 0
	for i := 0; i < len(ta); i++ {
		for j := 0; j < len(tb); j++ {
			n := 0
			for i+n < len(ta) && j+n < len(tb) && ta[i+n] == tb[j+n] {
				n++
			}
			if n > best {
				best = n
			}
		}
	}
	return best
}

// substringScore rewards one slug embedding the other
// ("opt-in-flow" inside "double-opt-in-flow"). Short embeddings are
// noise, so only embeddings at least 8 chars count.
func substringScore(a, b string) float64 {
	if len(a) >= 8 && strings.Contains(b, a) {
		return 0.9
	}
	if len(b) >= 8 && strings.Contains(a, b) {
		return 0.9
	}
	return 0
}

// editDistanceScore is 1 minus the Levenshtein distance normalized by
// the longer slug.
func editDistanceScore(a, b string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	return 1 - float64(levenshtein(a, b))/float64(maxLen)
}

// levenshtein computes the edit distance between two slugs. Slugs are
// validated ASCII, so byte-wise comparison is exact.
func levenshtein(a, b string) int {
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			m := prev[j] + 1 // deletion
			if v := curr[j-1] + 1; v < m {
				m = v // insertion
			}
			if v := prev[j-1] + cost; v < m {
				m = v // substitution
			}
			curr[j] = m
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
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
