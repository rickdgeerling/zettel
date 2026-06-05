# Zettel MCP

A Go CLI and MCP server for managing a Zettelkasten memory system. Designed as long-term memory for AI agents.

## Installation

```bash
go install
```

## CLI Usage

```bash
# Read a card
zettel read my-card

# Write a card (stdin)
cat card.md | zettel write my-card

# Write with --body flag
zettel write my-card --body "$(cat card.md)"

# Search
zettel search "jwt"

# Search with filters
zettel search "token" --tag security --category backend

# List link graph
zettel links

# Per-card links
zettel links my-card

# Archive
zettel archive my-card

# Start MCP server
zettel serve
```

## MCP Server

Add to your MCP client config (e.g., Claude Desktop):

```json
{
  "mcpServers": {
    "zettel": {
      "command": "zettel",
      "args": ["serve"]
    }
  }
}
```

Available tools:
- `zettel_search` — search with optional query, tag, category, status filters
- `zettel_read` — read a card by slug
- `zettel_write` — create/update a card
- `zettel_links` — global link graph or per-card links
- `zettel_archive` — move a card to archive

## Storage

Cards are stored as Markdown files in `~/.zettel/cards/`. Archived cards are in `~/.zettel/archived/`.

## Development

```bash
go test ./...        # run all tests
go test ./internal/store/... -cover  # check coverage
go build ./...       # build
```

## Inspiration

This project is heavily inspired by [Memex](https://github.com/iamtouchskyer/memex), it's SKILL files were the implementation guide.
