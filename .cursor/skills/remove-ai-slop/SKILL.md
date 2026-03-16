---
name: remove-ai-slop
description: Remove AI-generated code artifacts and unnecessary defensive patterns. Use when cleaning AI-style code in a branch or preparing code for final review.
---

# Remove AI code slop

Clean up AI-generated code artifacts in the current branch while preserving behavior.

## Workflow

1. Check the branch diff against `main`.
2. Identify and remove AI-style artifacts introduced in this branch.
3. Keep changes aligned with local file conventions and surrounding code style.
4. Preserve intended logic and avoid unrelated refactors.

## What to clean

- Extra comments that are inconsistent with file style or add no real value.
- Unnecessary defensive checks or `try/catch` blocks that are abnormal for trusted/validated call paths.
- `any` casts used to bypass real type issues.
- Deeply nested code that should be simplified with guard clauses and early returns.
- Other style inconsistencies that clearly do not match the touched area.

## Guardrails

- Do not change behavior unless the existing behavior is clearly accidental AI artifact.
- Do not remove validation or checks that are required for security boundaries.
- Keep imports at the top of files.
- Keep edits focused and minimal.

## Output

At the end, report only a 1-3 sentence summary of what was changed.
