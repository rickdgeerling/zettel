/**
 * zettel Extension for Pi
 *
 * Persistent Zettelkasten memory for AI coding agents.
 * Wraps the `zettel` CLI as Pi custom tools so the LLM can recall prior knowledge
 * at task start and save insights at task end.
 *
 * Prerequisites: go install github.com/rickdgeerling/zettel
 *
 * Tools registered:
 *   zettel_recall  – retrieve memory index or search (call at task start)
 *   zettel_retro   – save an atomic insight card (call at task end)
 *   zettel_search  – full-text search cards (with status/category/date filters)
 *   zettel_read    – read a specific card by slug
 *   zettel_write   – write/update a card
 *   zettel_links   – show link graph stats or broken links
 *   zettel_organize – emit an organize report (orphans, hubs, conflicts, broken links)
 *   zettel_archive – archive a card
 *
 * Session lifecycle:
 *   before_agent_start  – injects recall reminder on first turn
 *   agent_end           – injects retro reminder if not yet done
 *   session_compact     – resets recall state so reminder re-injects post-compaction
 *   resources_discover  – exposes bundled skills (zettel-recall, zettel-retro, etc.)
 */

import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { Type } from "@sinclair/typebox";
import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const baseDir = dirname(fileURLToPath(import.meta.url));
const skillsDir = join(baseDir, "skills");

/** Run a zettel CLI command and return stdout. Uses spawn to support stdin. */
function zettel(
  args: string[],
  stdin?: string,
): Promise<{ stdout: string; stderr: string; ok: boolean }> {
  return new Promise((resolve) => {
    const child = spawn("zettel", [...args, "--quiet"], { timeout: 30_000 });

    let stdout = "";
    let stderr = "";

    child.stdout.on("data", (data: Buffer) => {
      stdout += data.toString();
    });
    child.stderr.on("data", (data: Buffer) => {
      stderr += data.toString();
    });

    child.on("error", (err: NodeJS.ErrnoException) => {
      if (err.code === "ENOENT") {
        resolve({
          stdout: "",
          stderr:
            "zettel CLI not found. Install it with: npm install -g @touchskyer/zettel",
          ok: false,
        });
      } else {
        resolve({ stdout: stdout.trim(), stderr: err.message, ok: false });
      }
    });

    child.on("close", (code: number | null) => {
      resolve({ stdout: stdout.trim(), stderr: stderr.trim(), ok: code === 0 });
    });

    if (stdin !== undefined) {
      child.stdin.write(stdin);
      child.stdin.end();
    }
  });
}

function textResult(text: string, isError = false) {
  return {
    content: [{ type: "text" as const, text }],
    details: {},
    ...(isError ? { isError: true } : {}),
  };
}

function appendStderrWarning(text: string, stderr: string): string {
  return stderr ? `${text}\n\n${stderr}` : text;
}

/** Coerce a tool parameter to a non-negative integer, falling back when absent/invalid. */
function toInt(v: unknown, fallback: number): number {
  const n = Number(v);
  return Number.isFinite(n) ? Math.floor(n) : fallback;
}

// ---------------------------------------------------------------------------
// Extension
// ---------------------------------------------------------------------------

export default function zettelExtension(pi: ExtensionAPI) {
  // Track whether recall/retro have been done this session
  let recallDone = false;
  let retroDone = false;

  // -----------------------------------------------------------------------
  // Resource discovery — expose bundled skills to Pi
  // -----------------------------------------------------------------------

  pi.on("resources_discover", async () => {
    if (!existsSync(skillsDir)) return;
    return {
      skillPaths: [skillsDir],
    };
  });

  // -----------------------------------------------------------------------
  // Session lifecycle
  // -----------------------------------------------------------------------

  pi.on("session_start", async (_event, _ctx) => {
    recallDone = false;
    retroDone = false;
  });

  // After compaction the recall reminder message is summarized away.
  // Reset recallDone so before_agent_start re-injects the reminder on
  // the next user prompt.
  pi.on("session_compact", async (_event, _ctx) => {
    recallDone = false;
  });

  // Inject recall reminder at the start of each agent turn (until recall is done)
  pi.on("before_agent_start", async (_event, _ctx) => {
    if (recallDone) return;

    return {
      message: {
        customType: "zettel-recall-reminder",
        content: [
          "## zettel Memory System Active",
          "",
          "You have a Zettelkasten memory system available via the `zettel_recall` tool.",
          "**BEFORE starting work**, call `zettel_recall` (with no query) to load your keyword index.",
          "Then read relevant cards with `zettel_read` and search with `zettel_search`.",
          "Never include actual secrets, credentials, tokens, or exact secret file contents in zettel query/body arguments; use abstract descriptions or redacted examples.",
          "",
          "**AFTER completing the task**, if you learned something non-obvious,",
          "call `zettel_retro` to save atomic insight cards for future sessions.",
          "",
          "Memory recall guardrails: max 3 link hops, max 20 cards read.",
        ].join("\n"),
        display: false,
      },
    };
  });

  // After each agent response, remind the LLM to call zettel_retro if it
  // hasn't done so yet. Uses "nextTurn" delivery so it doesn't interrupt
  // the current response — the reminder appears on the next user prompt.
  pi.on("agent_end", async (_event, _ctx) => {
    if (retroDone || !recallDone) return;

    pi.sendMessage(
      {
        customType: "zettel-retro-reminder",
        content: [
          "**zettel reminder:** If you learned something non-obvious in this task,",
          "call `zettel_retro` to save an atomic insight card before finishing.",
        ].join(" "),
        display: false,
      },
      {
        deliverAs: "nextTurn",
      },
    );
  });

  // -----------------------------------------------------------------------
  // Tool: zettel_recall
  // -----------------------------------------------------------------------

  pi.registerTool({
    name: "zettel_recall",
    label: "zettel Recall",
    description:
      "IMPORTANT: Call at the START of every task. Retrieves your persistent Zettelkasten memory — knowledge cards from previous sessions. Returns the keyword index (if exists) or card list. USAGE: Call with NO query first to get the index. Only use query when you need to find specific cards — pass 1-3 short keywords, NOT sentences or task summaries. Keyword search uses AND logic (every token must appear in the same card). For natural-language search, use zettel_search with semantic=true instead. Never include actual secrets, credentials, tokens, or exact secret file contents in query.",
    parameters: Type.Object({
      query: Type.Optional(
        Type.String({
          description:
            "1-3 short keywords (AND logic — every token must appear). Do NOT pass sentences or task summaries. Omit for task-start recall. Examples: 'pptx migration', 'auth gotcha'. Do not include raw secrets.",
        }),
      ),
    }),
    async execute(_toolCallId, params) {
      recallDone = true;
      const { query } = params as { query?: string };

      if (query) {
        const res = await zettel(["search", query, "--limit", "10"]);
        if (!res.ok) return textResult(res.stderr, true);
        return textResult(
          appendStderrWarning(res.stdout || "No cards found.", res.stderr),
        );
      }

      // Try index first, then fall back to listing all cards
      const indexRes = await zettel(["read", "index"]);
      if (indexRes.ok && indexRes.stdout) {
        return textResult(indexRes.stdout);
      }

      const listRes = await zettel(["search", "--limit", "10"]);
      if (!listRes.ok) return textResult(listRes.stderr, true);
      return textResult(
        listRes.stdout || "No cards yet. This is a fresh memory.",
      );
    },
  });

  // -----------------------------------------------------------------------
  // Tool: zettel_retro
  // -----------------------------------------------------------------------

  pi.registerTool({
    name: "zettel_retro",
    label: "zettel Retro",
    description:
      "IMPORTANT: Call at the END of every task to save what you learned. Write one atomic insight per card with [[wikilinks]] to related cards. Only save non-obvious learnings. Never save actual secrets, credentials, tokens, or exact secret file contents. Handles frontmatter automatically.",
    parameters: Type.Object({
      slug: Type.String({
        description: "Card slug in kebab-case (e.g. 'jwt-revocation-pattern')",
      }),
      title: Type.String({
        description: "Card title (≤60 chars, noun phrase not sentence)",
      }),
      body: Type.String({
        description:
          "Card body in markdown with [[wikilinks]] explaining relationships",
      }),
      category: Type.Optional(
        Type.String({
          description: "Category (e.g. frontend, architecture, devops)",
        }),
      ),
      tags: Type.Optional(
        Type.Array(
          Type.String({
            description: "Tags (e.g. gotcha, decision, docker)",
          }),
        ),
      ),
    }),
    async execute(_toolCallId, params) {
      const { slug, title, body, category } = params as {
        slug: string;
        title: string;
        body: string;
        category?: string;
      };

      const today = new Date().toISOString().split("T")[0];
      const frontmatter: Record<string, string> = {
        title,
        created: today,
        source: "pi",
      };
      if (category) frontmatter.category = category;

      const yaml = Object.entries(frontmatter)
        .map(([k, v]) => `${k}: ${v}`)
        .join("\n");
      const content = `---\n${yaml}\n---\n\n${body}`;

      const res = await zettel(["write", slug], content);
      if (!res.ok) return textResult(res.stderr, true);
      retroDone = true;
      return textResult(
        appendStderrWarning(`Card '${slug}' saved successfully.`, res.stderr),
      );
    },
  });

  // -----------------------------------------------------------------------
  // Tool: zettel_search
  // -----------------------------------------------------------------------

  pi.registerTool({
    name: "zettel_search",
    label: "zettel Search",
    description:
      "Full-text search memory cards. Omit query to list all cards. Never include actual secrets, credentials, tokens, or exact secret file contents in query.",
    parameters: Type.Object({
      query: Type.Optional(
        Type.String({
          description:
            "Search keyword (substring match on title/body/tags/category)",
        }),
      ),
      status: Type.Optional(
        Type.String({
          description: "Filter by frontmatter status (e.g. 'conflict')",
        }),
      ),
      category: Type.Optional(
        Type.String({ description: "Filter by frontmatter category" }),
      ),
      modified_since: Type.Optional(
        Type.String({
          description: "Only cards modified on or after this date (YYYY-MM-DD)",
        }),
      ),
      limit: Type.Optional(
        Type.Number({ description: "Max results (default 10, 0 = unlimited)" }),
      ),
      offset: Type.Optional(
        Type.Number({
          description: "Skip N results for pagination (default 0)",
        }),
      ),
    }),
    async execute(_toolCallId, params) {
      const p = params as {
        query?: string;
        status?: string;
        category?: string;
        modified_since?: string;
        limit?: number;
        offset?: number;
      };
      const args = ["search"];
      if (p.query) args.push(p.query);
      if (p.status) args.push("--status", p.status);
      if (p.category) args.push("--category", p.category);
      if (p.modified_since) args.push("--modified-since", p.modified_since);
      // Always pass offset and limit explicitly so pagination is deterministic.
      args.push("--offset", String(toInt(p.offset, 0)));
      args.push("--limit", String(toInt(p.limit, 10)));

      const res = await zettel(args);
      if (!res.ok) return textResult(res.stderr, true);
      return textResult(
        appendStderrWarning(res.stdout || "No cards found.", res.stderr),
      );
    },
  });

  // -----------------------------------------------------------------------
  // Tool: zettel_read
  // -----------------------------------------------------------------------

  pi.registerTool({
    name: "zettel_read",
    label: "zettel Read",
    description: "Read a specific memory card by its slug.",
    parameters: Type.Object({
      slug: Type.String({ description: "Card slug (e.g. 'my-card-name')" }),
    }),
    async execute(_toolCallId, params) {
      const { slug } = params as { slug: string };
      const res = await zettel(["read", slug]);
      if (!res.ok) return textResult(res.stderr, true);
      return textResult(res.stdout);
    },
  });

  // -----------------------------------------------------------------------
  // Tool: zettel_write
  // -----------------------------------------------------------------------

  pi.registerTool({
    name: "zettel_write",
    label: "zettel Write",
    description:
      "Write or update a memory card. Content should include YAML frontmatter + markdown body. Never write actual secrets, credentials, tokens, or exact secret file contents.",
    parameters: Type.Object({
      slug: Type.String({
        description: "Card slug in kebab-case",
      }),
      content: Type.String({
        description:
          "Full card content: YAML frontmatter (---) + markdown body with [[wikilinks]]",
      }),
    }),
    async execute(_toolCallId, params) {
      const { slug, content } = params as { slug: string; content: string };
      const res = await zettel(["write", slug], content);
      if (!res.ok) return textResult(res.stderr, true);
      return textResult(
        appendStderrWarning(`Card '${slug}' written successfully.`, res.stderr),
      );
    },
  });

  // -----------------------------------------------------------------------
  // Tool: zettel_links
  // -----------------------------------------------------------------------

  pi.registerTool({
    name: "zettel_links",
    label: "zettel Links",
    description:
      "Show link graph stats. Omit slug for global stats, or specify a slug for that card's links. Set broken=true to list wikilinks pointing to missing/archived cards.",
    parameters: Type.Object({
      slug: Type.Optional(
        Type.String({
          description: "Card slug (omit for global stats)",
        }),
      ),
      broken: Type.Optional(
        Type.Boolean({
          description:
            "List [[wikilinks]] pointing to missing or archived cards",
        }),
      ),
      include_archived: Type.Optional(
        Type.Boolean({
          description: "Include archived cards in the global graph",
        }),
      ),
    }),
    async execute(_toolCallId, params) {
      const { slug, broken, include_archived } = params as {
        slug?: string;
        broken?: boolean;
        include_archived?: boolean;
      };
      const args = ["links"];
      if (broken) {
        args.push("--broken");
      } else if (include_archived) {
        args.push("--include-archived");
      }
      if (slug && !broken) args.push(slug);

      const res = await zettel(args);
      if (!res.ok) return textResult(res.stderr, true);
      return textResult(res.stdout || "No link data.");
    },
  });

  // -----------------------------------------------------------------------
  // Tool: zettel_organize
  // -----------------------------------------------------------------------

  pi.registerTool({
    name: "zettel_organize",
    label: "zettel Organize",
    description:
      "Emit an organize report: orphans, hubs, unresolved conflicts, recently modified cards (with their linked neighbors), and broken links. Optional since date (YYYY-MM-DD) limits the recently-modified section.",
    parameters: Type.Object({
      since: Type.Optional(
        Type.String({
          description:
            "Only include recently modified cards since this date (YYYY-MM-DD)",
        }),
      ),
    }),
    async execute(_toolCallId, params) {
      const { since } = params as { since?: string };
      const args = ["organize"];
      if (since) args.push("--since", since);

      const res = await zettel(args);
      if (!res.ok) return textResult(res.stderr, true);
      return textResult(res.stdout);
    },
  });

  // -----------------------------------------------------------------------
  // Tool: zettel_archive
  // -----------------------------------------------------------------------

  pi.registerTool({
    name: "zettel_archive",
    label: "zettel Archive",
    description:
      "Archive a card (move to archive). Use for outdated or superseded cards. Idempotent.",
    parameters: Type.Object({
      slug: Type.String({ description: "Card slug to archive" }),
      dry_run: Type.Optional(
        Type.Boolean({
          description: "Show what would be archived without changing anything",
        }),
      ),
    }),
    async execute(_toolCallId, params) {
      const { slug, dry_run } = params as { slug: string; dry_run?: boolean };
      const args = ["archive", slug];
      if (dry_run) args.push("--dry-run");
      const res = await zettel(args);
      if (!res.ok) return textResult(res.stderr, true);
      return textResult(res.stdout || `Card '${slug}' archived.`);
    },
  });

  // -----------------------------------------------------------------------
  // Tool: zettel_store_path
  // -----------------------------------------------------------------------

  pi.registerTool({
    name: "zettel_store_path",
    label: "zettel store-path",
    description: "Fetch the filesystem path where the zettel cards live",
    parameters: {},
    async execute() {
      const res = await zettel(["store-path"]);
      if (!res.ok) return textResult(res.stderr, true);
      return textResult(
        appendStderrWarning(`Zettel store-path: ${res.stdout}`, res.stderr),
      );
    },
  });
}
