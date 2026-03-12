---
name: grafana-frontend-performance
description: Builds and reviews performant frontend components that align with Grafana repository conventions. Use when creating, refactoring, or reviewing React/TypeScript UI code in Grafana, especially when the user mentions performance, rendering, component architecture, or style consistency.
---

# Grafana Frontend Performance

## Purpose

Produce frontend component changes that are fast, maintainable, and consistent with this repository's patterns.

## Use This Skill When

- The task touches React components, hooks, or rendering behavior
- The user asks for performance improvements in UI code
- The task requires matching Grafana frontend conventions
- The task includes component refactors, new UI componentry, or feature UI expansion

## Core Standards

- Prefer function components with hooks.
- Follow existing local patterns before introducing new abstractions.
- Keep business logic out of presentation-only components.
- Use Emotion styling patterns already used in Grafana (`useStyles2` and related style helpers).
- Prefer Redux Toolkit and existing slice patterns for global state.
- Prefer RTK Query patterns for data fetching where that is already the local convention.

## Performance Defaults

- Minimize avoidable re-renders:
  - Keep props stable where practical.
  - Avoid creating new objects/functions in hot render paths unless needed.
  - Memoize expensive derived values with `useMemo` only when profiling or clear cost justifies it.
- Keep component boundaries intentional:
  - Split large components when state/updates can be isolated.
  - Keep frequently updating state close to the consumers.
- Avoid unnecessary effects:
  - Do not use `useEffect` for pure derivation that can happen during render.
  - Ensure effect dependencies are complete and stable.
- Load only what is needed:
  - Defer heavy UI/code paths when appropriate.
  - Avoid broad imports when narrower imports exist in the local pattern.

## Styling And UX Conventions

- Match spacing, typography, color, and interaction patterns from nearby Grafana UI code.
- Reuse existing shared components from `@grafana/ui` before creating new primitives.
- Preserve accessibility basics:
  - Keyboard reachability
  - Semantic structure
  - Meaningful labels and names

## Implementation Workflow

Use this checklist for each task:

- [ ] Identify the nearest existing feature/component pattern and follow it.
- [ ] Implement with clear component boundaries and minimal render churn.
- [ ] Reuse existing Grafana UI and styling primitives where possible.
- [ ] Add or update tests for behavior impacted by the change.
- [ ] Run focused validation commands for modified files.

## Validation Commands

Run the smallest relevant set first, then broaden if needed:

```bash
yarn test path/to/changed.test.tsx
yarn lint
yarn typecheck
```

## Output Expectations

When delivering changes, include:

1. What performance-sensitive decisions were made
2. Which repo conventions were followed (component, styling, state, data fetching)
3. What tests and checks were run
