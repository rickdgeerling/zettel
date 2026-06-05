/**
 * Project Tasks Extension
 *
 * Project-specific build, test, and format tools for zettel-mcp.
 */

import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";

export default function (pi: ExtensionAPI) {
  pi.registerTool({
    name: "build",
    label: "Build",
    description: "Compile the zettel binary",
    promptSnippet: "Build the zettel binary",
    parameters: Type.Object({}),
    async execute(_toolCallId, _params, _signal, _onUpdate, ctx) {
      const result = await pi.exec("go", ["build", "-o", "zettel", "./"], {
        cwd: ctx.cwd,
      });
      return {
        content: [
          {
            type: "text",
            text: result.stdout || result.stderr || `exit code ${result.code}`,
          },
        ],
        details: { exitCode: result.code },
      };
    },
  });

  pi.registerTool({
    name: "test",
    label: "Test",
    description: "Run the Go test suite",
    promptSnippet: "Run Go tests",
    parameters: Type.Object({
      packages: Type.Optional(
        Type.Array(Type.String(), {
          description: "Package paths to test (defaults to ./...)",
        }),
      ),
    }),
    async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
      const args = ["test"];
      if (params.packages && params.packages.length > 0) {
        args.push(...params.packages);
      } else {
        args.push("./...");
      }
      const result = await pi.exec("go", args, {
        cwd: ctx.cwd,
      });
      return {
        content: [
          {
            type: "text",
            text: result.stdout || result.stderr || `exit code ${result.code}`,
          },
        ],
        details: { exitCode: result.code },
      };
    },
  });

  pi.registerTool({
    name: "format",
    label: "Format",
    description: "Format Go source files with gofmt",
    promptSnippet: "Format Go source with gofmt",
    parameters: Type.Object({
      files: Type.Optional(
        Type.Array(Type.String(), {
          description: "Source file paths to format (defaults to ./)",
        }),
      ),
    }),
    async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
      const args = ["-w"];
      if (params.files && params.files.length > 0) {
        args.push(...params.files);
      } else {
        args.push("./");
      }
      const result = await pi.exec("gofmt", args, {
        cwd: ctx.cwd,
      });
      return {
        content: [
          {
            type: "text",
            text: result.stdout || result.stderr || `exit code ${result.code}`,
          },
        ],
        details: { exitCode: result.code },
      };
    },
  });
}
