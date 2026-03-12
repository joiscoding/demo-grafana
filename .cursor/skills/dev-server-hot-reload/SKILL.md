---
name: dev-server-hot-reload
description: Start Grafana development servers with hot reloading for both backend and frontend. Use when the user wants to run the dev server, start development with hot reload, or build the app with live reloading enabled.
---

# Development Server with Hot Reloading

## Quick Start

Start both backend and frontend development servers with hot reloading:

1. **Backend**: Run `make run` in one terminal
2. **Frontend**: Run `yarn start:liveReload` in a separate terminal

## Workflow

The backend (`make run`) uses Air for hot reloading and watches for Go file changes. The frontend (`yarn start:liveReload`) runs the webpack dev server with live reload enabled.

Both processes should run simultaneously in separate terminal sessions. The backend typically runs on port 3000, and the frontend dev server proxies to it.

## Notes

- Do not run these commands unless explicitly requested by the user
- Both commands run indefinitely until stopped (Ctrl+C)
- Ensure dependencies are installed (`make deps`) before starting