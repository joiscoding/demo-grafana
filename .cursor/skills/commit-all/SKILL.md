---
name: commit-all
description: Reads git status and diffs, buckets by feature, and outputs short one-line commit messages like [feat] or [docs]. Use when the user asks to commit-all, plan commits, or split a dirty tree into commits.
---

# Commit-all: plan commits from a dirty tree

Turn a mixed working tree into a **small number of coherent commits**—without committing yet unless the user asks.

## 1. Gather state

Run (repo root):

```bash
git status -sb
git diff --stat
git diff          # unstaged
git diff --cached # staged
```

If the change set is huge, use `git diff --stat` per path and spot-read `git diff -- path1 path2` for each bucket you form.

**Do not** stage or commit secrets, env files, or unrelated personal artifacts. Call out anything that should stay untracked or go in a separate PR.

## 2. Bucket changes (feature-first)

**Primary axis: one bucket = one feature, product behavior, or owned area**—not “all docs” vs “all TS.” Ask: *what shipped story does this belong to?*

1. **Name the feature** — e.g. “alert rule export,” “architecture onboarding docs,” “Explore query bar,” “CI for plugin X.” Each bucket is that slice of work end-to-end.
2. **Keep the slice together** — code + tests + feature-specific docs + config that exists *because* of that feature stay in the **same** commit when they’re small and reviewable as one unit. Reverting the commit should undo one capability.
3. **Split when stories differ** — two user-visible behaviors or two unrelated areas → two buckets, even if both touch `pkg/` or both touch `public/app/`.
4. **Cross-cutting only when nothing else fits** — repo-wide refactors, mass renames, or pure infra (no feature owner) get their own bucket; don’t hide unrelated features inside “chore.”

| If you see… | Bucket as… |
|-------------|------------|
| Same flow (UI → API → store) | **One feature commit** including its tests and user-facing docs |
| Feature A + Feature B in one branch | **Two commits** by feature, not by frontend/backend |
| Docs that only describe Feature X | **With Feature X** unless the doc dump is huge (then doc commit immediately after, same PR) |
| Generated output for Feature X | **With Feature X** unless policy says otherwise |
| Unrelated fixes snuck in | **Separate commit** per fix / per feature |

Merge only when the work is truly one story; split when revert or review would conflate unrelated features.

## 3. Propose commits

**One line per commit only**—no multi-line blurbs. Format:

`[type] lowercase short imperative (≤ ~60 chars)`

Types: `[feat]`, `[fix]`, `[docs]`, `[chore]`, `[refactor]`, `[test]`—pick one per commit.

Rules:

- Single sentence after the bracket; no scope parentheses, no Conventional Commits colon form.
- Order lines so dependencies make sense (e.g. base change before follow-up).
- **Do not** add “What / Paths / Staging” blocks in the default answer. If the user asks to run commits, give a minimal `git add … && git commit -m "…"` per line using the **exact** one-liner as `-m`.

Example (this is the whole proposal):

```text
1. [docs] add architecture overview and backend split
2. [feat] add council multi-agent skill
3. [feat] add onboard developer skill
```

## 4. After the plan

- If the user wants execution: stage per bucket, commit with `-m "[type] …"` matching the one-liner exactly.
- If something doesn’t fit: suggest **WIP commit**, **drop**, or **separate PR** explicitly.

## Agent instructions

1. Bucket by **feature / user story / product area** first; file type is secondary.
2. Prefer **one revert-friendly unit per commit** (one feature or one fix), with its tests and feature docs bundled when reasonable.
3. Base buckets on **diff intent and behavior**, not directory layout alone.
4. Mention **binary/generated** files and keep them with the feature that required them unless repo convention says otherwise.
5. **Output**: numbered list of `[type] …` one-liners only unless the user asks for staging help.
