---
name: demo-reset
description: Reset the demo environment to its starting state by reverting all source code changes while preserving demo setup files. Use when the user says "reset the demo", "start fresh", "demo reset", "clean slate", or wants to prepare for another demo run.
---

# Demo Reset

Resets the Grafana demo repo to its clean starting state so the demo can be run again from scratch. All source code fixes made during the demo are reverted. Demo infrastructure (skills, scripts, prompts, agents) is preserved.

## What Gets Reset

| Area | What Happens | Why |
|------|-------------|-----|
| `pkg/` source files | Restored to `main` branch state | Reverts logging fixes (Issue #12) and SQL/ES error handling fixes (Issue #10) |
| Uncommitted changes | Discarded | Cleans up any in-progress work |
| `.cursor/plans/` | All `.plan.md` files deleted | Removes plans created during the demo so Plan Mode starts fresh |

## What Is Preserved

- `.cursor/skills/` -- all skills including this one
- `.cursor/agents/` -- council reviewer agent definitions
- `DEMO_SCRIPT.md` and `DEMO_PROMPTS.md`
- All committed demo setup on the `mpotteiger/demo-changes` branch

## Reset Steps

Run these commands sequentially:

```sh
# 1. Discard all uncommitted changes (staged and unstaged)
git checkout -- .
git clean -fd --exclude=.cursor/

# 2. Restore pkg/ to main branch state (undo any committed demo fixes)
git checkout main -- pkg/

# 3. Remove any plan files created during the demo
rm -f .cursor/plans/*.plan.md

# 4. Stage the revert and amend the latest commit
git add -A
git commit --amend --no-edit

# 5. Force push the clean state
git push --force-with-lease
```

After running these steps, confirm the reset by running:

```sh
# Should show no pkg/ differences vs main
git diff main...HEAD --stat -- pkg/

# Should show clean working tree
git status
```

## Important Notes

- This is safe to force push because `mpotteiger/demo-changes` is a personal branch.
- The reset is idempotent -- running it multiple times is safe.
- After reset, remind the user to **close all Cursor chat tabs** manually so stale conversation context doesn't carry over into the demo.
