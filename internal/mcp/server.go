package mcp

import (
	"context"
	"fmt"
	"sort"
	"strings"

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
		z.makeLinksTool(),
		z.makeArchiveTool(),
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
	case "zettel_links":
		return z.handleLinks(request.Params.Arguments)
	case "zettel_archive":
		return z.handleArchive(request.Params.Arguments)
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

func (z *ZettelServer) makeLinksTool() server.ServerTool {
	tool := mcp.NewTool("zettel_links",
		mcp.WithDescription("Show link graph stats (global summary or per-card links)"),
		mcp.WithString("slug", mcp.Title("Slug"), mcp.Description("Card slug (optional, omit for global graph)")),
	)
	return server.ServerTool{
		Tool:    tool,
		Handler: z.handleTool,
	}
}

func (z *ZettelServer) makeArchiveTool() server.ServerTool {
	tool := mcp.NewTool("zettel_archive",
		mcp.WithDescription("Move a card from active to archived"),
		mcp.WithString("slug", mcp.Title("Slug"), mcp.Description("Card slug to archive"), mcp.Required()),
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
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
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

	var tags []string
	if t, ok := argsMap["tag"].(string); ok && t != "" {
		tags = []string{t}
	}

	var category, status *string
	if c, ok := argsMap["category"].(string); ok && c != "" {
		category = &c
	}
	if s, ok := argsMap["status"].(string); ok && s != "" {
		status = &s
	}

	limit, offset := getIntArg(argsMap, "limit"), getIntArg(argsMap, "offset")

	results, err := z.store.SearchCards(query, tags, category, status, limit, offset)
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

	if limit > 0 && len(results) == limit {
		lines = append(lines, fmt.Sprintf("(showing %d results, use offset %d for more)", limit, offset+limit))
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

	slug, hasSlug := argsMap["slug"].(string)
	if !hasSlug || slug == "" {
		return z.handleGlobalLinks()
	}
	return z.handleCardLinks(slug)
}

func (z *ZettelServer) handleGlobalLinks() (*mcp.CallToolResult, error) {
	graph, err := z.store.LinkGraph()
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
		if stats.Inbound == 0 {
			marker = " (ORPHAN)"
			orphans++
		} else if stats.Inbound >= 10 {
			marker = " (HUB)"
			hubs++
		}
		line := fmt.Sprintf("%s: %d inbound, %d outbound%s", s, stats.Inbound, stats.Outbound, marker)
		lines = append(lines, line)
	}

	summary := fmt.Sprintf("\nSummary: %d cards, %d orphans, %d hubs", len(graph), orphans, hubs)
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

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: msg}},
		IsError: true,
	}
}
