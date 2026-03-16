---
name: grafana-storybook
description: >-
  Starts Grafana @grafana/ui Storybook on localhost without the Go backend.
  Use when the user wants frontend-only UI, Storybook, design system preview,
  component stories, or to run Grafana UI without make run / backend.
---

# Grafana Storybook (frontend-only)

Runs **Storybook** for `@grafana/ui`: real Grafana React components in isolation. **Not** the full Grafana app (no login, dashboards API, or Explore)—that still requires `make run` + `yarn start`.

## When to use

- Preview or develop **@grafana/ui** components and stories
- **No Go / no backend**—Node + Yarn only
- Default URL: **http://localhost:9001**

## Instructions

When the user asks to run Storybook, frontend-only UI, or Grafana UI without backend, run from the **repo root** using the Shell tool (background if long-running).

### Preflight

1. Check terminals folder; if Storybook is already on port `9001`, say so instead of starting a duplicate.
2. If `node_modules` is missing or Storybook fails on missing deps:

```sh
corepack enable && corepack install   # if yarn not on PATH
yarn install --immutable
```

If `yarn` is unavailable, use `npx yarn` instead of `yarn` below.

### Start Storybook

```sh
yarn workspace @grafana/ui storybook
```

This runs `storybook dev -p 9001` (see `packages/grafana-ui/package.json`). **`--no-open`** is set in that script—tell the user to open **http://localhost:9001** manually.

Run in background (`block_until_ms: 0`). Wait until logs show Storybook is ready (e.g. compiled / listening on 9001).

### Optional root script

```sh
yarn storybook
```

Root `package.json` invokes the same workspace with `--ci`—prefer **`yarn workspace @grafana/ui storybook`** for interactive local dev unless CI-style is intended.

## Notes

- Port **9001** conflicts: stop the other process or free the port before starting.
- **Playwright Storybook e2e** uses this server (`yarn e2e:playwright:storybook`); do not conflate with full Grafana e2e.
- Full product UI remains **http://localhost:3000** with backend + `yarn start` (see `start-dev-server` skill).
