package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rickdgeerling/zettel/internal/store"
)

type ZettelServer struct {
	srv   *server.MCPServer
	store *store.Store
}

func NewZettelServer(store *store.Store) *ZettelServer {
	z := &ZettelServer{store: store}
	z.srv = server.NewMCPServer("zettel", "1.0.0")

	z.srv.AddTools(
		z.makeSearchTool(),
		z.makeReadTool(),
		z.makeWriteTool(),
		z.makeAppendTool(),
		z.makeLinksTool(),
		z.makeArchiveTool(),
		z.makeOrganizeTool(),
		z.makeIndexTool(),
	)

	return z
}

func (z *ZettelServer) Run() error {
	return server.ServeStdio(z.srv)
}

func (z *ZettelServer) handleTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	switch request.Params.Name {
	case "zettel_search":
		return z.handleSearch(request.Params.Arguments)
	case "zettel_read":
		return z.handleRead(request.Params.Arguments)
	case "zettel_write":
		return z.handleWrite(request.Params.Arguments)
	case "zettel_append":
		return z.handleAppend(request.Params.Arguments)
	case "zettel_links":
		return z.handleLinks(request.Params.Arguments)
	case "zettel_archive":
		return z.handleArchive(request.Params.Arguments)
	case "zettel_organize":
		return z.handleOrganize(request.Params.Arguments)
	case "zettel_index":
		return z.handleIndex()
	default:
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Unknown tool: " + request.Params.Name}},
			IsError: true,
		}, nil
	}
}

func (z *ZettelServer) makeSearchTool() server.ServerTool {
	tool := mcp.NewTool("zettel_search",
		mcp.WithDescription("Search cards by substring query with optional metadata filters and pagination"),
		mcp.WithString("query", mcp.Title("Query"), mcp.Description("Substring search query"), mcp.Required()),
		mcp.WithString("tag", mcp.Title("Tag"), mcp.Description("Filter by tag (single)")),
		mcp.WithString("category", mcp.Title("Category"), mcp.Description("Filter by category")),
		mcp.WithString("status", mcp.Title("Status"), mcp.Description("Filter by status")),
		mcp.WithString("modified_since", mcp.Title("Modified Since"), mcp.Description("Only cards modified on or after this date (YYYY-MM-DD)")),
		mcp.WithNumber("limit", mcp.Title("Limit"), mcp.Description("Max results to return (0 = unlimited)")),
		mcp.WithNumber("offset", mcp.Title("Offset"), mcp.Description("Skip N results (for pagination)")),
	)
	return server.ServerTool{
		Tool:    tool,
		Handler: z.handleTool,
	}
}

func (z *ZettelServer) makeReadTool() server.ServerTool {
	tool := mcp.NewTool("zettel_read",
		mcp.WithDescription("Read a card by slug (checks active and archived)"),
		mcp.WithString("slug", mcp.Title("Slug"), mcp.Description("Card slug"), mcp.Required()),
	)
	return server.ServerTool{
		Tool:    tool,
		Handler: z.handleTool,
	}
}

func (z *ZettelServer) makeWriteTool() server.ServerTool {
	tool := mcp.NewTool("zettel_write",
		mcp.WithDescription("Write a card (creates or overwrites)"),
		mcp.WithString("slug", mcp.Title("Slug"), mcp.Description("Card slug"), mcp.Required()),
		mcp.WithString("body", mcp.Title("Body"), mcp.Description("Card content with YAML frontmatter"), mcp.Required()),
	)
	return server.ServerTool{
		Tool:    tool,
		Handler: z.handleTool,
	}
}

func (z *ZettelServer) makeAppendTool() server.ServerTool {
	tool := mcp.NewTool("zettel_append",
		mcp.WithDescription("Append a line to a card's section without touching existing prose (creates the section if absent). The safe primitive for append-only edits such as adding - [[slug]]: relationship entries to ## Links. modified is bumped."),
		mcp.WithString("slug", mcp.Title("Slug"), mcp.Description("Card slug"), mcp.Required()),
		mcp.WithString("text", mcp.Title("Text"), mcp.Description("Line to append, e.g. '- [[other-slug]]: why related'"), mcp.Required()),
		mcp.WithString("section", mcp.Title("Section"), mcp.Description("Section heading to append to (default: Links; created if absent)")),
	)
	return server.ServerTool{
		Tool:    tool,
		Handler: z.handleTool,
	}
}

func (z *ZettelServer) handleAppend(args any) (*mcp.CallToolResult, error) {
	argsMap, ok := args.(map[string]any)
	if !ok {
		return errorResult("invalid arguments"), nil
	}

	slug, ok := argsMap["slug"].(string)
	if !ok || slug == "" {
		return errorResult("slug is required"), nil
	}

	text, ok := argsMap["text"].(string)
	if !ok || text == "" {
		return errorResult("text is required"), nil
	}

	section := "Links"
	if s, ok := argsMap["section"].(string); ok && s != "" {
		section = s
	}

	if err := z.store.AppendToSection(slug, section, text); err != nil {
		return errorResult(fmt.Sprintf("Append failed: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Appended to %q section of card %q", section, slug)}},
	}, nil
}

func (z *ZettelServer) makeLinksTool() server.ServerTool {
	tool := mcp.NewTool("zettel_links",
		mcp.WithDescription("Show link graph stats (global summary or per-card links), or broken links"),
		mcp.WithString("slug", mcp.Title("Slug"), mcp.Description("Card slug (optional, omit for global graph)")),
		mcp.WithBoolean("broken", mcp.Title("Broken"), mcp.Description("List [[wikilinks]] pointing to missing or archived cards")),
		mcp.WithBoolean("include_archived", mcp.Title("Include Archived"), mcp.Description("Include archived cards in the global graph")),
	)
	return server.ServerTool{
		Tool:    tool,
		Handler: z.handleTool,
	}
}

func (z *ZettelServer) makeArchiveTool() server.ServerTool {
	tool := mcp.NewTool("zettel_archive",
		mcp.WithDescription("Move a card from active to archived (idempotent)"),
		mcp.WithString("slug", mcp.Title("Slug"), mcp.Description("Card slug to archive"), mcp.Required()),
	)
	return server.ServerTool{
		Tool:    tool,
		Handler: z.handleTool,
	}
}

func (z *ZettelServer) makeOrganizeTool() server.ServerTool {
	tool := mcp.NewTool("zettel_organize",
		mcp.WithDescription("Emit an organize report: orphans, hubs, unresolved conflicts, recently modified cards, and broken links"),
		mcp.WithString("since", mcp.Title("Since"), mcp.Description("Only include recently modified cards since this date (YYYY-MM-DD)")),
	)
	return server.ServerTool{
		Tool:    tool,
		Handler: z.handleTool,
	}
}

func getIntArg(argsMap map[string]any, key string) int {
	v, ok := argsMap[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case float32:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case int32:
		return int(n)
	case uint:
		return int(n)
	case uint64:
		return int(n)
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i)
		}
		if f, err := n.Float64(); err == nil {
			return int(f)
		}
	case string:
		// Some MCP clients/harnesses serialize numbers as strings.
		if f, err := strconv.ParseFloat(n, 64); err == nil {
			return int(f)
		}
	}
	return 0
}

func getBoolArg(argsMap map[string]any, key string) bool {
	v, ok := argsMap[key]
	if !ok {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case string:
		parsed, err := strconv.ParseBool(b)
		return err == nil && parsed
	}
	return false
}

func parseDateArg(argsMap map[string]any, key string) (time.Time, error) {
	v, ok := argsMap[key].(string)
	if !ok || v == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", v)
}

func (z *ZettelServer) handleSearch(args any) (*mcp.CallToolResult, error) {
	argsMap, ok := args.(map[string]any)
	if !ok {
		return errorResult("invalid arguments"), nil
	}

	query, _ := argsMap["query"].(string)
	if query == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "No query provided. Use zettel_search with a query string."}},
		}, nil
	}

	opts := store.SearchOptions{
		Query:  query,
		Limit:  getIntArg(argsMap, "limit"),
		Offset: getIntArg(argsMap, "offset"),
	}

	if t, ok := argsMap["tag"].(string); ok && t != "" {
		opts.Tags = []string{t}
	}
	if c, ok := argsMap["category"].(string); ok && c != "" {
		opts.Category = c
	}
	if st, ok := argsMap["status"].(string); ok && st != "" {
		opts.Status = st
	}
	if since, err := parseDateArg(argsMap, "modified_since"); err != nil {
		return errorResult(fmt.Sprintf("Invalid modified_since date (use YYYY-MM-DD): %v", err)), nil
	} else {
		opts.ModifiedSince = since
	}

	results, err := z.store.Search(opts)
	if err != nil {
		return errorResult(fmt.Sprintf("Search failed: %v", err)), nil
	}

	if len(results) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "No cards found"}},
		}, nil
	}

	var lines []string
	for _, r := range results {
		tagsStr := ""
		if len(r.Tags) > 0 {
			tagsStr = " [" + strings.Join(r.Tags, ", ") + "]"
		}
		line := fmt.Sprintf("%s | %s%s | cat=%s | status=%s | created=%s modified=%s",
			r.Slug, r.Title, tagsStr, r.Category, r.Status, r.Created, r.Modified)
		lines = append(lines, line)
	}

	if opts.Limit > 0 && len(results) == opts.Limit {
		lines = append(lines, fmt.Sprintf("(showing %d results, use offset %d for more)", opts.Limit, opts.Offset+opts.Limit))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: strings.Join(lines, "\n")}},
	}, nil
}

func (z *ZettelServer) handleRead(args any) (*mcp.CallToolResult, error) {
	argsMap, ok := args.(map[string]any)
	if !ok {
		return errorResult("invalid arguments"), nil
	}

	slug, ok := argsMap["slug"].(string)
	if !ok || slug == "" {
		return errorResult("slug is required"), nil
	}

	card, err := z.store.ReadCard(slug)
	if err != nil {
		return errorResult(fmt.Sprintf("Card not found: %s", slug)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: card.Marshal()}},
	}, nil
}

func (z *ZettelServer) handleWrite(args any) (*mcp.CallToolResult, error) {
	argsMap, ok := args.(map[string]any)
	if !ok {
		return errorResult("invalid arguments"), nil
	}

	slug, ok := argsMap["slug"].(string)
	if !ok || slug == "" {
		return errorResult("slug is required"), nil
	}

	body, ok := argsMap["body"].(string)
	if !ok || body == "" {
		return errorResult("body is required"), nil
	}

	card, err := store.UnmarshalCard(slug, []byte(body))
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid card format: %v", err)), nil
	}

	err = z.store.WriteCard(slug, card, "mcp")
	if err != nil {
		return errorResult(fmt.Sprintf("Write failed: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Card %q written successfully", slug)}},
	}, nil
}

func (z *ZettelServer) handleLinks(args any) (*mcp.CallToolResult, error) {
	argsMap, ok := args.(map[string]any)
	if !ok {
		return errorResult("invalid arguments"), nil
	}

	if getBoolArg(argsMap, "broken") {
		return z.handleBrokenLinks()
	}

	slug, hasSlug := argsMap["slug"].(string)
	if !hasSlug || slug == "" {
		return z.handleGlobalLinks(getBoolArg(argsMap, "include_archived"))
	}
	return z.handleCardLinks(slug)
}

func (z *ZettelServer) handleBrokenLinks() (*mcp.CallToolResult, error) {
	broken, err := z.store.BrokenLinks()
	if err != nil {
		return errorResult(fmt.Sprintf("BrokenLinks failed: %v", err)), nil
	}

	if len(broken) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "No broken links"}},
		}, nil
	}

	var lines []string
	for _, b := range broken {
		lines = append(lines, fmt.Sprintf("%s -> [[%s]] (%s)", b.From, b.To, b.Kind))
	}
	lines = append(lines, fmt.Sprintf("\nSummary: %d broken links", len(broken)))

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Broken Links:\n" + strings.Join(lines, "\n")}},
	}, nil
}

func (z *ZettelServer) handleGlobalLinks(includeArchived bool) (*mcp.CallToolResult, error) {
	var graph map[string]store.LinkStats
	var err error
	if includeArchived {
		graph, err = z.store.LinkGraphIncludingArchived()
	} else {
		graph, err = z.store.LinkGraph()
	}
	if err != nil {
		return errorResult(fmt.Sprintf("LinkGraph failed: %v", err)), nil
	}

	var lines []string
	var orphans, hubs int

	var slugs []string
	for s := range graph {
		slugs = append(slugs, s)
	}
	sort.Strings(slugs)

	for _, s := range slugs {
		stats := graph[s]
		marker := ""
		if stats.Archived {
			marker = " (ARCHIVED)"
		} else if stats.Inbound == 0 {
			marker = " (ORPHAN)"
			orphans++
		} else if stats.Inbound >= 10 {
			marker = " (HUB)"
			hubs++
		}
		line := fmt.Sprintf("%s: %d inbound, %d outbound%s", s, stats.Inbound, stats.Outbound, marker)
		lines = append(lines, line)
	}

	summary := fmt.Sprintf("\nSummary: %d active cards, %d orphans, %d hubs", len(graph), orphans, hubs)
	if includeArchived {
		summary = fmt.Sprintf("\nSummary: %d cards (including archived), %d orphans, %d hubs", len(graph), orphans, hubs)
	}
	lines = append(lines, summary)

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Link Graph:\n" + strings.Join(lines, "\n")}},
	}, nil
}

func (z *ZettelServer) handleCardLinks(slug string) (*mcp.CallToolResult, error) {
	inbound, outbound, err := z.store.LinksForCard(slug)
	if err != nil {
		return errorResult(fmt.Sprintf("Card not found: %s", slug)), nil
	}

	text := fmt.Sprintf("Card: %s\n  Inbound (%d): %v\n  Outbound (%d): %v",
		slug, len(inbound), inbound, len(outbound), outbound)

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: text}},
	}, nil
}

func (z *ZettelServer) handleArchive(args any) (*mcp.CallToolResult, error) {
	argsMap, ok := args.(map[string]any)
	if !ok {
		return errorResult("invalid arguments"), nil
	}

	slug, ok := argsMap["slug"].(string)
	if !ok || slug == "" {
		return errorResult("slug is required"), nil
	}

	err := z.store.ArchiveCard(slug)
	if err != nil {
		return errorResult(fmt.Sprintf("Archive failed: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Card %q archived successfully", slug)}},
	}, nil
}

func (z *ZettelServer) handleOrganize(args any) (*mcp.CallToolResult, error) {
	argsMap, ok := args.(map[string]any)
	if !ok {
		return errorResult("invalid arguments"), nil
	}

	since, err := parseDateArg(argsMap, "since")
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid since date (use YYYY-MM-DD): %v", err)), nil
	}

	report, err := z.store.OrganizeReport(since)
	if err != nil {
		return errorResult(fmt.Sprintf("Organize report failed: %v", err)), nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Organize Report\nTotal active cards: %d", report.Total))

	lines = append(lines, fmt.Sprintf("\nOrphans (%d):", len(report.Orphans)))
	if len(report.Orphans) == 0 {
		lines = append(lines, "  (none)")
	} else {
		for _, slug := range report.Orphans {
			lines = append(lines, "  "+slug)
		}
	}

	lines = append(lines, fmt.Sprintf("\nHubs (%d):", len(report.Hubs)))
	if len(report.Hubs) == 0 {
		lines = append(lines, "  (none)")
	} else {
		for _, slug := range report.Hubs {
			lines = append(lines, "  "+slug)
		}
	}

	lines = append(lines, fmt.Sprintf("\nUnresolved Conflicts (%d):", len(report.Conflicts)))
	if len(report.Conflicts) == 0 {
		lines = append(lines, "  (none)")
	} else {
		for _, slug := range report.Conflicts {
			lines = append(lines, "  "+slug)
		}
	}

	recentHeader := fmt.Sprintf("\nRecently Modified (%d):", len(report.RecentlyModified))
	if !since.IsZero() {
		recentHeader = fmt.Sprintf("\nRecently Modified since %s (%d):", since.Format("2006-01-02"), len(report.RecentlyModified))
	}
	lines = append(lines, recentHeader)
	if len(report.RecentlyModified) == 0 {
		lines = append(lines, "  (none)")
	} else {
		for _, rc := range report.RecentlyModified {
			lines = append(lines, fmt.Sprintf("  %s (modified %s) — out: %s | in: %s",
				rc.Slug, rc.Modified, markNeighbors(rc.Outbound, "→"), markNeighbors(rc.Inbound, "←")))
		}
	}

	lines = append(lines, fmt.Sprintf("\nBroken Links (%d):", len(report.BrokenLinks)))
	if len(report.BrokenLinks) == 0 {
		lines = append(lines, "  (none)")
	} else {
		for _, b := range report.BrokenLinks {
			if b.Suggestion != "" {
				lines = append(lines, fmt.Sprintf("  %s -> [[%s]] (%s, did you mean [[%s]]?)", b.From, b.To, b.Kind, b.Suggestion))
			} else {
				lines = append(lines, fmt.Sprintf("  %s -> [[%s]] (%s)", b.From, b.To, b.Kind))
			}
		}
	}

	// Record the run date for the next incremental run.
	if err := z.store.StampLastOrganize(); err != nil {
		return errorResult(fmt.Sprintf("Failed to stamp last-organize date: %v", err)), nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: strings.Join(lines, "\n")}},
	}, nil
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: msg}},
		IsError: true,
	}
}

func (z *ZettelServer) makeIndexTool() server.ServerTool {
	tool := mcp.NewTool("zettel_index",
		mcp.WithDescription("Emit a draft keyword index: every active card grouped by frontmatter category with titles as placeholder descriptions. Edit the descriptions and category placement, then save with zettel_write (slug 'index'). Turns index maintenance from authoring into editing."),
	)
	return server.ServerTool{
		Tool:    tool,
		Handler: z.handleTool,
	}
}

func (z *ZettelServer) handleIndex() (*mcp.CallToolResult, error) {
	draft, err := z.store.DraftIndex()
	if err != nil {
		return errorResult(fmt.Sprintf("DraftIndex failed: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: draft}},
	}, nil
}

// markNeighbors renders a neighbor list with direction markers (→ out,
// ← in).
func markNeighbors(slugs []string, marker string) string {
	if len(slugs) == 0 {
		return "(none)"
	}
	parts := make([]string, len(slugs))
	for i, s := range slugs {
		parts[i] = marker + s
	}
	return strings.Join(parts, ", ")
}
