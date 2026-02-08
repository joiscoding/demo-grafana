---
name: council-review
description: Multi-agent code review using a committee of experts pattern. Invokes parallel custom subagents pinned to different models across labs to review changes from multiple angles, then synthesizes findings. Use when the user asks for a council review, committee review, multi-agent review, or wants changes reviewed from multiple perspectives.
---

# Council of Agents Review

A multi-agent review pattern that leverages model diversity across labs. Instead of one model reviewing its own work, spawn specialized reviewer subagents -- each pinned to a different model -- to examine changes from independent angles.

## Why Cross-Lab Review

Models tend to favor code patterns similar to what they generate. A model reviewing its own output is like a writer editing their own work -- blind spots persist. By reviewing with models from different labs (OpenAI, Google) than the one that wrote the code (Anthropic), you get genuinely independent perspectives. Different training data, different reasoning patterns, different blind spots.

## The Review Council

Four custom subagents are defined in `.cursor/agents/`, each with a pinned model:

| Subagent | Model | Lab | Focus |
|---|---|---|---|
| `/correctness-reviewer` | `gpt-5.2` | OpenAI | Logic errors, bugs, API misuse |
| `/completeness-reviewer` | `gpt-5.2-codex` | OpenAI | Missed instances in the broader codebase |
| `/edge-case-reviewer` | `gemini-3-pro` | Google | Regressions, failure modes, error paths |
| `/style-reviewer` | `composer` | Fast | Naming, conventions, codebase consistency |

All subagents are `readonly: true` -- they analyze but don't modify code.

## Instructions

### Step 1: Identify the Changes

Determine what to review. The user will provide one of:
- Specific files or directories that changed
- A git diff reference
- A description of the changes

If unclear, ask: "Which files or changes should the council review?"

### Step 2: Gather Context

Before invoking reviewers, read the changed files and build a concise brief:
- What changed and why
- Which files were modified
- The patterns or conventions involved

This brief gets passed to each subagent so they have full context.

### Step 3: Invoke All Four Reviewers in Parallel

Launch all four subagents **in a single message** so they run concurrently. Use the `/name` syntax or reference them by name. Pass each one:
- The change summary
- The list of files to review
- Any relevant context about the codebase conventions

Example invocation:
```
Invoke these four subagents in parallel with the following context:

Changes: [SUMMARY]
Files: [FILE_LIST]

1. /correctness-reviewer - Review for logical errors and bugs
2. /completeness-reviewer - Search for missed instances
3. /edge-case-reviewer - Analyze failure modes and regressions
4. /style-reviewer - Check convention adherence
```

### Step 4: Synthesize Findings

After all four subagents return, synthesize into a single unified review:

```markdown
## Council Review Summary

**Scope:** [what was reviewed]
**Models:** gpt-5.2, gpt-5.2-codex, gemini-3-pro, composer (reviewing code written by Claude Opus)
**Reviewers:** Correctness, Completeness, Edge Cases, Style

### Critical Issues
[Must-fix items from any reviewer. If none: "None found -- all reviewers agree the changes are sound."]

### Suggestions
[Improvement ideas, tagged with which reviewer raised them.]

### Validation
[What the reviewers confirmed is correct. Convergence across models = high confidence.]

### Verdict
[One-sentence assessment: ship it, needs fixes, or needs rework.]
```

Highlight when multiple reviewers from different labs converge on the same conclusion -- that's the strongest signal.

## Example Usage

```
Council review the structured logging changes in pkg/api/ and pkg/services/
```

The agent will:
1. Read the changed files
2. Build a change brief
3. Invoke all 4 reviewer subagents in parallel (o3, gpt-4o, gemini-2.5-pro)
4. Synthesize a unified review with cross-lab confidence signals

## Model Rationale

- **gpt-5.2** (correctness): OpenAI's strongest reasoning model -- best for catching logic errors
- **gpt-5.2-codex** (completeness): Optimized for code search and comprehension
- **gemini-3-pro** (edge cases): Google's perspective catches what OpenAI misses
- **composer** (style): Fast model -- style checking is pattern-matching, doesn't need heavy reasoning
