---
description: AutoFlow Run - Execute current step
argument-hint: [optional details]
allowed-tools: Read, Glob, Grep, Bash, Task, Skill, WebSearch, WebFetch, AskUserQuestion
---

Execute the AutoFlow Run workflow for the current step.

Read and follow:
- `.claude/skills/tr/SKILL.md`
- `.claude/skills/tr/references/flow.md`

Templates: `.claude/skills/tr/templates/`

Input: `$ARGUMENTS` (optional execution details)

Do not modify files directly; delegate all file operations to Codex via `/file-op`.
