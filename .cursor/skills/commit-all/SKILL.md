---
name: commit-all
description: Reads git status and diffs, buckets by feature, and outputs short one-line commit messages like [feat] or [docs]. Use when the user asks to commit-all, plan commits, or split a dirty tree into commits.
---

# Commit-all

Group a dirty tree into a few coherent commits (plan only unless the user asks to run them).

**State:** `git status -sb`, `git diff --stat`, `git diff`, `git diff --cached`. Huge trees: `--stat` per path, then `git diff -- <paths>` per bucket. No secrets/env/personal junk; flag untracked or separate-PR stuff. Binaries/generated stay with the feature that needs them.

**Buckets:** By feature or product area—not file type. One story per bucket (code + tests + that feature’s docs/config when it’s one reviewable unit). Split unrelated behaviors; lump only true cross-cutting (mass rename, pure infra).

| Situation | Bucket |
|-----------|--------|
| One vertical slice | One commit (+ tests/docs for that slice) |
| Two features | Two commits |
| Docs/generated for feature X | With X (or doc-only commit after if huge) |
| Drive-by fixes | Own commit(s) |

**Messages:** Numbered lines only by default: `[feat|fix|docs|chore|refactor|test] short imperative` (~≤60 chars), one sentence, no scopes/colons. Order by dependency. On request: `git add … && git commit -m "…"` per line, exact `-m`. Edge cases: WIP, drop, or separate PR.

Example:

```text
1. [docs] add architecture overview and backend split
2. [feat] add council multi-agent skill
```
