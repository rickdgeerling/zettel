---
name: zettel-organize
description: Periodic maintenance of the Zettelkasten card network.
whenToUse: When the user asks to organize, maintain, or clean up their memory cards. Also when running scheduled maintenance to detect orphans, hubs, contradictions, and rebuild the keyword index.
---

# Memory Organize

You are maintaining a Zettelkasten memory system. Your job is to keep the card network healthy.

## Tools Available

Two equivalent interfaces exist — use whichever your environment supports:

| CLI (Claude Code with zettel in PATH) | MCP tool (VSCode / Cursor / any MCP client) |
| ------------------------------------- | ------------------------------------------- |
| `zettel organize [--since <date>]`    | `zettel_organize` with optional since arg   |
| `zettel links --broken`               | `zettel_links` with broken=true             |
| `zettel links`                        | `zettel_links` with no args                 |
| `zettel links <slug>`                 | `zettel_links` with slug arg                |
| `zettel index`                        | `zettel_index`                              |
| `zettel search <query>`               | `zettel_search` with query arg              |
| `zettel search --status <s>`          | `zettel_search` with status arg             |
| `zettel search --modified-since <d>`  | `zettel_search` with modified_since arg     |
| `zettel read <slug>`                  | `zettel_read` with slug arg                 |
| `zettel write <slug>`                 | `zettel_write` with slug arg and body       |
| `zettel append <slug> --text <line>`  | `zettel_append` with slug/text/section args |
| `zettel archive <slug>`               | `zettel_archive` with slug arg              |
| `zettel store_path`                   | `zettel_store_path`                         |

The rest of this skill uses CLI syntax for brevity. Substitute MCP tool calls if CLI is unavailable.

## Process

```dot
digraph organize {
    "Organize report" -> "Detect orphans (in=0)";
    "Detect orphans (in=0)" -> "For each orphan: zettel read + search related";
    "For each orphan: zettel read + search related" -> "LLM decides: append links / mark stale / leave alone";
    "Organize report" -> "Detect hubs (in >= 10)";
    "Detect hubs (in >= 10)" -> "LLM decides: split into smaller cards or leave alone";
    "Organize report" -> "Recently modified pairs";
    "Recently modified pairs" -> "Genuinely contradicts?" [shape=diamond];
    "Genuinely contradicts?" -> "Mark status: conflict, surface to human" [label="yes"];
    "Genuinely contradicts?" -> "Evolution?" [label="no"];
    "Evolution?" -> "Merge + archive old card" [label="yes"];
    "Evolution?" -> "Leave alone" [label="no"];
    "Organize report" -> "Resolve existing conflicts";
    "Resolve existing conflicts" -> "Ask user which belief wins";
    "All checks done" -> "Rebuild keyword index";
}
```

1. Run `zettel organize` to get the organize report. It emits orphans, hubs, unresolved conflicts, recently modified cards (paired with their inbound/outbound neighbors), and broken links in one shot. The run also stamps `.last-organize` with the run date automatically (see Incremental Strategy).
2. **Orphan detection**: For cards with 0 inbound links (the "Orphans" section — navigation cards like `index` and `*-index` are excluded by design):
   - `zettel read` the orphan
   - `zettel search` for potentially related cards
   - Decide: append a link from a related card to this orphan (`zettel append <related-slug> --text '- [[orphan-slug]]: relationship'`), or leave alone if truly standalone
3. **Hub detection**: For cards with >= 10 inbound links (the "Hubs" section):
   - `zettel read` the hub and its linkers
   - Decide: is this card too broad? Should it be split into smaller atomic concepts?
4. **Contradiction/staleness detection**: The "Recently Modified" section lists recently modified cards with their neighbors, direction-marked: `→ slug` are cards this card links to (outbound), `← slug` are cards linking to it (inbound). For each pair:
   - Read both cards' content (`zettel read` for full content if needed)
   - Determine: do these cards **genuinely contradict** each other, or is one simply an **evolution** of the other?
   - **If evolution** (new info supersedes old): merge the cards — append the old card's unique content to the new one, then `zettel archive` the old card
   - **If genuine contradiction** (two conflicting beliefs, unclear which is correct): mark the newer card with `status: conflict` in frontmatter and add a note explaining the contradiction. Do NOT auto-resolve — the human decides which belief wins
   - **If no conflict**: leave both cards alone
5. **Resolve existing conflicts**: Check the "Unresolved Conflicts" section. For each card with `status: conflict`:
   - Surface it to the user: "Card X conflicts with card Y — which perspective should we keep?"
   - If the user resolves it, remove `status: conflict` from frontmatter and merge/archive as directed
6. **Fix broken links**: Check the "Broken Links" section. Each `[[slug]]` pointing to a missing or archived card may carry a `did you mean [[other-slug]]?` suggestion — the nearest existing card by name similarity. Verify the suggestion actually matches the intended concept before re-pointing, then:
   - If the target is archived and the reference is stale, edit the referring card to remove or re-point the link
   - If the target is missing (never existed), remove the link or create the card
7. **Rebuild keyword index** (always, as last step)

## Keyword Index Maintenance

The keyword index (`index` card) is a curated concept → card mapping, inspired by Luhmann's Schlagwort register. It is the primary entry point for the recall skill.

After completing all checks, draft the index:

1. Run `zettel index` (or `zettel_index`) — it emits a draft index card with every active card grouped by its frontmatter `category`, titles as placeholder descriptions. Navigation cards are excluded; an existing index's `created` date is preserved
2. Edit the draft using judgment: rewrite placeholder descriptions into accurate one-liners, move cards between categories where the frontmatter category is wrong
3. Save the result with `zettel write index`

```markdown
---
title: Keyword Index
created: <original creation date>
source: organize
---

## <Concept Category 1>

- [[slug-a]] — one-line description
- [[slug-b]] — one-line description

## <Concept Category 2>

- [[slug-c]] — one-line description
```

Rules for the index:

- Each card should appear under 1-2 categories (not more)
- Categories should be meaningful concepts, not arbitrary groupings
- Descriptions should be one line, explaining what the card is about (not just the title)
- Archived cards must be removed from the index
- New cards should already be in the index (added at creation time by `zettel-retro`). Verify this — if any cards are missing, add them.
- If the index grows too large (150+ notes), consider splitting into sub-index for one or more of the larger categories. The sub-index is named `<category>-index` and follows the same template as the main index.

Use `zettel write index` to save the index.

## Incremental Strategy

1. Read last run date: `cat $(zettel store-path --quiet)/.last-organize 2>/dev/null`
2. If the file exists, only process cards where frontmatter `modified` >= that date, plus their linked neighbors. Query them directly:
   ```
   zettel search --modified-since <last-run-date>
   ```
3. If the file doesn't exist, this is the first run — process all cards
4. After the run, no manual stamping is needed: `zettel organize` (CLI and MCP) writes today's date to `$(zettel store-path --quiet)/.last-organize` automatically when it emits the report

To find unresolved conflicts without grepping the filesystem, use:
```
zettel search --status conflict
```

When the organize skill creates new cards (e.g., splitting hubs), use `source: organize` in the frontmatter.

## Operation Rules

- **Append only**: When adding links to existing cards, use `zettel append <slug> --section Links --text '- [[link-to-note]]: short explanation of relationship'` (or `zettel_append`). It inserts the line at the end of the `## Links` section — creating the section if absent — without rewriting any existing prose. Never modify existing prose with full-card rewrites when an append will do.
  - **Exception — post-archive cleanup**: After archiving cards, it is OK to remove or re-point `[[links]]` in active cards that now point only at archived/deleted cards. This keeps the network from accumulating dead references. Limit edits to `## Links` sections and stale `[[...]]` references; do not rewrite prose.
- **Merge**: Read both cards, append source content to target card via `zettel write`, then `zettel archive` the source.
- **Archive**: Use `zettel archive <slug>` to move stale cards out of active search. Use `zettel archive --dry-run <slug>` to preview. Archiving is idempotent — re-archiving an already-archived card is a safe no-op.
- **Be conservative**: When in doubt, leave cards alone. It's better to under-organize than to break useful connections.
