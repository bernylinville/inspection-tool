## Codex Supervisor Policy (executor=codex+opencode)

When `executor=codex+opencode`, Codex is the supervisor and must decide what to delegate:

Delegate to OpenCode ONLY when:
1) Repo-changing work is required (apply patches, write files, repo-mutating commands), OR
2) The task is simple but would be token-heavy for Codex (large mechanical edits, repetitive changes, big scaffolds).

Otherwise, Codex should do the work itself (read/search/analyze/plan/review) and keep outputs compact.

Ask OpenCode to return:
- `changedFiles`
- `diffSummary`
- `commands` (include exit codes)
- `notes`

Constraints:
- Do NOT paste long logs; summarize and include only what is needed to verify correctness.
- Prefer short diff summaries; list exact file paths changed.

## Gemini Delegation (Large Code Analysis)

When to delegate to Gemini via gask:
1) Large codebase analysis (>5 files or >1000 lines total)
2) Project structure understanding
3) Architecture analysis and dependency mapping
4) Cross-file pattern recognition

Ask Gemini to return:
- `summary`: High-level overview
- `structure`: Key components and relationships
- `insights`: Design patterns and recommendations
