# Zettel MCP

A Go CLI and MCP server for managing a Zettelkasten memory system, as an experiment for long-term memory for AI agents.

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

# Initialize a store in the current directory
zettel init

# Show resolved store path
zettel store-path

# Search
zettel search "jwt"

# Search with filters
zettel search "token" --tag security --category backend

# Search by status or modification date
zettel search --status conflict
zettel search --modified-since 2026-06-16

# List link graph (active cards only)
zettel links

# Include archived cards in the graph
zettel links --include-archived

# List broken links (wikilinks pointing to missing/archived cards)
zettel links --broken

# Per-card links
zettel links my-card

# Organize report (orphans, hubs, conflicts, recently modified, broken links)
zettel organize
zettel organize --since 2026-06-16

# Archive (idempotent)
zettel archive my-card
zettel archive --dry-run my-card

# Start MCP server
zettel serve

# Common flags
zettel --store /path/to/store search "token"     # use specific store
zettel --quiet read my-card                       # suppress store path log
```

## MCP Server

Add to your MCP client config (e.g., Claude Desktop):

```json
{
  "mcpServers": {
    "zettel": {
      "command": "zettel",
      "args": ["--quiet", "serve"]
    }
  }
}
```

Available tools:
- `zettel_search` — search with optional query, tag, category, status, and modified_since filters
- `zettel_read` — read a card by slug
- `zettel_write` — create/update a card
- `zettel_links` — global link graph, per-card links, or broken links
- `zettel_organize` — organize report (orphans, hubs, conflicts, recently modified, broken links)
- `zettel_archive` — move a card to archive (idempotent)

## Storage

Cards are stored as Markdown files with YAML frontmatter. Store discovery follows this precedence:

1. `--store` flag (e.g., `--store /path/to/store`) — explicit override
2. Tree-walk from current directory — finds nearest `.zettel/` directory (like Git)
3. `ZETTEL_HOME` environment variable — fallback when no local store exists

Initialize a store in your project root with `zettel init`:
```bash
cd my-project
zettel init   # creates .zettel/cards/ and .zettel/archived/
```

To use a global store as a fallback (like the old `~/.zettel` behavior), set:
```bash
export ZETTEL_HOME=$HOME/.zettel
```

The `--quiet` flag suppresses the store path log on stderr:
```bash
zettel --quiet store-path   # prints path only
```

Find the resolved store path:
```bash
zettel store-path   # e.g., /home/user/projects/my-app/.zettel
```

## Development

```bash
go test ./...        # run all tests
go test ./internal/store/... -cover  # check coverage
go build ./...       # build
```

## Inspiration

This project is heavily inspired by [Memex](https://github.com/iamtouchskyer/memex), it's SKILL files were the implementation guide.
