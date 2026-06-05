# AGENTS.md

## Project

Zettel MCP — a Go CLI + MCP server for managing a Zettelkasten memory system. Designed as long-term memory for AI agents.

## Architecture

**Shared Core + Thin Wrappers**

- `internal/store/` — all business logic: card CRUD, search, link graph, frontmatter parsing
- `cmd/` — Cobra CLI subcommands, thin wrappers that delegate to `internal/store`
- `internal/mcp/` — MCP server over stdio, thin wrappers that delegate to `internal/store`

Both CLI and MCP use the same `*store.Store` — no duplication, no roundtrip overhead.

## Build & Test

```bash
gofmt -w ./                    # format source files
go build -o zettel ./...       # compile
go test ./...                  # all tests
go test ./internal/store/ -cover  # store coverage
bash test/cli_integration.sh      # end-to-end CLI test
```

## Key Conventions

### Tests
- Store tests use `t.TempDir()` — no mocking, no shared fixtures
- Each test calls `store.Init(tmpDir)` directly to get a `*Store`
- Tests are a mix of flat top-level functions and `t.Run` subtests
- No table-driven tests (yet)

### Store Discovery
- `store.Init(root)` creates a store at any path (creates `cards/` and `archived/` subdirs)
- The old `DefaultStore()` hardcoded `$HOME/.zettel` — being replaced by tree-walk discovery
- New precedence: `--store` flag > walk up from CWD > `ZETTEL_HOME` env var

### Card Format
- Markdown files with YAML frontmatter: `title`, `created`, `modified`, `source`, `tags`, `category`, `status`
- `[[wikilink]]` syntax for cross-references (kebab-case slugs)
- `slug` is the filename (minus `.md`), validated: lowercase alphanumeric + hyphens, 2-60 chars

### CLI
- `cmd/root.go` has a `PersistentPreRun` that initializes the store
- Subcommands access the store via `GetStore()` (package-level variable in `cmd/`)
- `serve` command was a latent bug: it ignored `GetStore()` and called `DefaultStore()` directly
- New commands must be registered in `init()` via `rootCmd.AddCommand(...)`

### Error Handling
- CLI: errors to stderr, `os.Exit(1)`
- MCP: errors as MCP error responses with descriptive messages
- Store functions return errors to callers — no silent logging-and-swallowing

## Dependencies

- `cobra` — CLI framework
- `mark3labs/mcp-go` — MCP server SDK
- `gopkg.in/yaml.v3` — YAML frontmatter parsing
- Everything else is stdlib

## Skills

The `skills/` directory contains LLM-readable instructions for AI agents using the zettel system:
- `zettel-retro` — save insights after completing tasks
- `zettel-organize` — periodic maintenance (orphan detection, hub splitting, keyword index)
- `zettel-recall` — search and retrieve relevant context
- `zettel-best-practices` — reference guide for card quality

Skills are documentation only — not embedded or loaded at runtime.
