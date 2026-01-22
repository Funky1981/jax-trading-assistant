# Serena: Local Code Intelligence and Offline Use Guide

## What Serena is (plain English)
Serena is a local code-intelligence backend that runs on your machine. It reads files from disk and uses language servers (the same kind of engines used by VS Code) to understand code structure: symbols, references, and safe edit boundaries. Serena can run as an MCP server, which lets tools like Codex call it for precise navigation and edits without sending entire files to a model.

## Why use it
- Lower token use: symbol lookups and targeted edits avoid large file reads.
- Safer edits: operations happen at symbol boundaries rather than line guessing.
- Local by default: file reads, index builds, and queries run on your machine.

## Core features
- Project indexing (builds a symbol catalog for fast lookups).
- Language server integration (Go, Python, TypeScript, Markdown, YAML in this repo).
- Semantic code tools (find symbol, find references, rename, insert/replace by symbol).
- File operations (read/write files, line edits, search for patterns).
- Memory store (named project notes you can read/write on demand).
- Onboarding and health check (indexing and tool validation).
- MCP server mode (exposes tools to external agents like Codex).

## How Serena works
1) You define a project in `.serena/project.yml` (languages, ignores, read-only mode).
2) Serena starts language servers for those languages and builds caches in `.serena/cache`.
3) Tools query the caches and language servers for symbols and references.
4) Only the minimum required file content is read for a task.

## Project files and folders
- `.serena/project.yml`: per-project config (languages, ignore rules, read-only).
- `.serena/cache/`: symbol index caches (one per language).
- `.serena/memories/`: named project notes (short, targeted context).
- `.serena/logs/`: logs from indexing and health checks.

## Language servers used here
These are local processes that parse code and answer symbol queries.
- Go: `gopls` (from the Go toolchain)
- Python: `pyright-langserver`
- TypeScript/JS: `typescript-language-server`
- YAML: `yaml-language-server`
- Markdown: `marksman`

They communicate over stdin/stdout and do not need the internet to answer queries.

## MCP usage (how Codex talks to Serena)
Serena can run as an MCP server. When enabled, Codex can call Serena tools such as:
- list directories
- search for patterns
- get symbols overview
- find symbol and references
- edit around symbol boundaries
- read/write memories

This keeps most code navigation local and cheap, and only surfaces the needed snippets.

## Token-saving workflow (recommended)
- Start with `list_dir`, `search_for_pattern`, or `get_symbols_overview`.
- Use `find_symbol` and `find_referencing_symbols` instead of opening files.
- Read only the specific file sections needed for an edit.
- Store stable context in `.serena/memories/` and load on demand.

## Closed environment (offline / approval-gated) use
The key idea is to pre-install everything once, with approvals, then run offline.

### External calls that may occur
These happen only during install or first-time setup:
- Serena package download (GitHub or PyPI via uvx).
- Node packages for language servers (npm).
- Marksman binary download (GitHub release).

Once installed, normal indexing and queries are local.

### How to set up for offline use
1) Pre-install Serena and language servers during an approved window.
2) Warm the caches by running `serena project index` once.
3) Freeze or copy the installed assets and caches into your offline image.
4) Run with offline flags (examples below) and no external registry access.

### Suggested offline controls
- Use internal mirrors/registries for PyPI and npm.
- Set environment flags to prevent downloads:
  - `UV_OFFLINE=1`
  - `UV_NO_PYTHON_DOWNLOADS=1`
  - `NPM_CONFIG_OFFLINE=true`
- Keep `.serena/project.yml` languages minimal to avoid new installs.

### What gets installed and where (Windows)
- Serena language servers:
  - `C:\Users\<user>\.serena\language_servers\static\...`
- Serena caches:
  - `C:\Projects\jax-trading assistant\.serena\cache\...`
- uv/pyright artifacts:
  - `C:\Users\<user>\AppData\Local\uv\cache\...`

Copy those into your offline image if you want zero downloads.

## Security considerations
- Serena reads local files you point it at; it does not exfiltrate by itself.
- If you expose Serena via MCP, ensure only trusted clients can connect.
- Keep read-only mode available for sensitive repos (`read_only: true`).
- Restrict shell execution if you do not want Serena to run commands.
- Treat `.serena/memories/` as internal documentation (no secrets).

## Health checks and logs
- Health check validates language servers and tools.
- Logs are written under `.serena/logs/`.
- If your console encoding is cp1252, emoji output may fail; logs still capture results.

## Summary
Serena is a local, language-server-backed tool that helps agents navigate and edit code with less token use. With a one-time, approved install, it can run fully offline. Most security risk is controllable by restricting network access and tool permissions.
