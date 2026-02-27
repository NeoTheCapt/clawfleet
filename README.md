# 🚢 Clawfleet

Distributed AI Agent Cluster Management Platform.

Deploy, orchestrate and manage AI agent teams (OpenClaw, ZeroClaw, Nanobot) across distributed nodes.

## Quick Start

```bash
# Start Control Plane
make run-server

# Start Node Agent (connects to local server)
make run-agent

# Build binaries
make build-all
# outputs to: dist/bin/
```

## Architecture

- **Control Plane** (`clawfleet-server`) — Central API, scheduling, monitoring
- **Node Agent** (`clawfleet-node`) — Runs on each node, manages agent containers
- **Agent Adapters** — Pluggable support for OpenClaw, ZeroClaw, Nanobot

## API

- `GET    /api/health` — Health check
- `GET    /api/version` — Version info (server + components)
- `GET    /api/nodes` — List nodes
- `GET    /api/agents` — List agent instances
- `POST   /api/agents` — Create agent instance
- `DELETE /api/agents/:id` — Delete agent (async, purges after container removed)
- `GET    /api/companies` — List companies
- `GET    /api/companies/:id/positions` — Org structure (departments/positions)
- `GET    /api/companies/:id/bots` — Telegram bots
- `POST   /api/companies/:id/bots` — Add Telegram bot
- `POST   /api/register` — Node registration
- `POST   /api/heartbeat` — Node heartbeat (includes agent version)
- `POST   /api/task/complete` — Node reports task result

## License

MIT
