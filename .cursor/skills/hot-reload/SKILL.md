---
name: hot-reload
description: Explains how hot reload works for frontend and backend development. Use when the user asks about hot reload, live reload, auto-refresh, file watching, why changes aren't appearing, or previewing the app in Cursor's built-in browser.
---

# Hot reload

## Overview

Grafana's local development environment supports automatic rebuilding for both the frontend and backend. This means code changes are picked up without manually restarting processes.

## Backend (Go) — Air

The backend uses [Air](https://github.com/air-verse/air), a live-reload utility for Go. Running `make run` starts Air with the config in `.air.toml`.

**What it watches:** Go, INI, TOML, HTML, and JSON files in `apps/`, `conf/`, `pkg/`, and `public/views/`.

**What it does:** When a watched file changes, Air rebuilds the binary (`make GO_BUILD_DEV=1 build-air`) and restarts the server automatically. The old process receives an interrupt signal before the new one starts.

**Key `.air.toml` settings:**

- `include_ext`: `["go", "ini", "toml", "html", "json"]`
- `include_dir`: `["apps", "conf", "pkg", "public/views"]`
- `exclude_regex`: `["_test.go", "_gen.go"]`
- `kill_delay`: 500ms grace period before killing the old process

## Frontend (TypeScript/React) — Webpack watch

The frontend uses webpack in watch mode. Running `yarn start` starts the watcher, which incrementally rebuilds when source files change.

**What it watches:** TypeScript and SCSS files in the app source, excluding `node_modules` and decoupled datasource plugins.

**What it does:** Webpack detects file changes and performs an incremental rebuild. The browser must be refreshed manually to pick up changes unless LiveReload is enabled.

### Enabling LiveReload (auto browser refresh)

To enable automatic browser refresh on frontend changes, start with:

```sh
yarn start:liveReload
```

This is equivalent to `yarn start -- --env liveReload=1`. It activates the `webpack-livereload-plugin`, which injects a script tag into the page that listens on port `35750` for rebuild notifications.

## Previewing in Cursor's built-in browser

After both dev servers are running and healthy, open Grafana inside Cursor using the `browser_navigate` MCP tool:

```
CallMcpTool: cursor-ide-browser / browser_navigate
  url: "http://localhost:3000"
```

To open side-by-side with code, set `position` to `"side"`:

```
CallMcpTool: cursor-ide-browser / browser_navigate
  url: "http://localhost:3000"
  position: "side"
```

After making code changes, reload the built-in browser instead of telling the user to refresh manually:

```
CallMcpTool: cursor-ide-browser / browser_reload
```

Use `browser_snapshot` or `browser_take_screenshot` to verify the UI reflects the change without requiring the user to check themselves.

## Troubleshooting

### Changes aren't appearing

1. **Backend changes:** Confirm Air is running (`make run`, not `make run-go`). Check the terminal for rebuild messages. If Air seems stuck, stop it and restart with `make run`.
2. **Frontend changes:** Confirm the webpack watcher is running (`yarn start`). Look for "Compiled successfully" after saving a file. Hard-refresh the browser (`Cmd+Shift+R`) if the change isn't showing.
3. **LiveReload not working:** Make sure you started with `yarn start:liveReload`. Check that port `35750` isn't blocked or in use by another process.

### Slow rebuilds

- Frontend: Use `yarn start -- --env noTsCheck=1` to skip TypeScript checking during watch, or `--env noLint=1` to skip ESLint. Both reduce rebuild time.
- Backend: Air excludes test and generated files by default. Large changes to `conf/` or `public/views/` can trigger rebuilds — this is expected.
