# Symbiont

A local-first Neptune Apex dashboard and data platform. Replaces Apex Fusion with a self-hosted stack — no cloud, no Neptune account required after initial setup.

---

## What It Does

- Polls your Neptune Apex controller every 10 seconds over the local network
- Stores all probe readings and outlet states in an embedded DuckDB database
- Serves a REST API and real-time SSE stream
- Provides a React dashboard with live charts, history, and outlet control
- Exposes an MCP server so AI assistants (Claude, etc.) can query and control your tank
- Includes a CLI for scripting and terminal access

---

## Requirements

- Neptune Apex running AOS 5+, accessible on the local network
- Linux (x86_64 or arm64) or macOS (Apple Silicon)
- Tailscale (optional, for remote access)

---

## Install

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/kjaebker/Symbiont/main/install.sh)
```

The installer prompts for your Apex controller IP, username, and password, then writes the config and (on Linux) installs a systemd service.

**Start the service:**

```bash
# Linux
sudo systemctl enable --now symbiont
sudo journalctl -u symbiont -f          # token is printed on first run — save it

# macOS
launchctl load ~/Library/LaunchAgents/com.symbiont.plist
tail -f ~/.symbiont/symbiont.log        # token is printed on first run — save it
```

Dashboard: **http://localhost:8420**

> The token is generated once on first start and printed to the log. It's needed to log in to the dashboard and for CLI/MCP use. If you lose it, see [Authentication](#authentication).

---

## Manual Install

If you prefer not to pipe to bash:

1. Download the right archive from the [latest release](https://github.com/kjaebker/Symbiont/releases/latest):
   - Linux x86_64: `symbiont-linux-amd64.tar.gz`
   - Linux arm64: `symbiont-linux-arm64.tar.gz`
   - macOS Apple Silicon: `symbiont-darwin-arm64.tar.gz`

2. Extract and install:

```bash
tar -xzf symbiont-*.tar.gz
sudo mv symbiont /usr/local/bin/symbiont
```

3. Configure (`.env.example` is included in the tarball):

```bash
sudo mkdir -p /etc/symbiont
sudo cp .env.example /etc/symbiont/env
sudo chmod 600 /etc/symbiont/env
sudo nano /etc/symbiont/env   # set SYMBIONT_APEX_URL, SYMBIONT_APEX_USER, SYMBIONT_APEX_PASS
```

4. Run:

```bash
# One-shot (foreground)
symbiont serve

# Or as a systemd service (Linux) — see install.sh for the unit file
```

---

## CLI Usage

```bash
# Current probe values
symbiont probes current

# Probe history
symbiont probes history Temp --interval 5m

# Outlet states and control
symbiont outlets list
symbiont outlets set <outlet-id> OFF
symbiont outlets events

# Alert rules
symbiont alerts list
symbiont alerts create
symbiont alerts update <id>
symbiont alerts delete <id>
symbiont alerts events

# Notification channels
symbiont notify list
symbiont notify create
symbiont notify delete <id>
symbiont notify test

# Display config (probe/outlet labels and ordering)
symbiont config probes list
symbiont config probes update <name>
symbiont config outlets list
symbiont config outlets update <id>

# System
symbiont system status
symbiont system backup
symbiont system cleanup
symbiont system backups
symbiont system log

# Token management
symbiont auth tokens list
symbiont auth tokens create --label "claude-desktop"
symbiont auth tokens revoke <id>
```

Add `--json` to any command for machine-readable output.

---

## AI Integration (MCP)

Symbiont exposes an MCP server that lets Claude and other AI assistants query and control your tank.

**Claude Desktop:** Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "symbiont": {
      "command": "/usr/local/bin/symbiont",
      "args": ["mcp"],
      "env": {
        "SYMBIONT_API_URL": "http://localhost:8420",
        "SYMBIONT_TOKEN": "your-token-here"
      }
    }
  }
}
```

**Claude Code:**

```bash
claude mcp add symbiont /usr/local/bin/symbiont mcp \
  --env SYMBIONT_API_URL=http://localhost:8420 \
  --env SYMBIONT_TOKEN=your-token
```

Available tools: `get_current_parameters`, `get_probe_history`, `get_outlet_states`, `control_outlet`, `get_outlet_event_log`, `get_alert_rules`, `get_alert_events`, `get_system_status`, `get_system_log`, `get_devices`, `summarize_tank_health`.

---

## Authentication

All API endpoints require a Bearer token. The token is generated on first run and printed once to the log. Additional tokens can be created via:

```bash
symbiont auth tokens create --label "phone"
```

Tokens are stored in SQLite. If the original token is lost:

```bash
symbiont auth reset --db-path /var/lib/symbiont/app.db --yes
```

---

## Data

- **DuckDB** (`telemetry.db`) — all probe readings and outlet states, 10-second granularity
- **SQLite** (`app.db`) — alert rules, display config, tokens, outlet event log, backup records

Default retention: 1 year. Configurable via `SYMBIONT_RETENTION_DAYS`. Cleanup runs weekly.

Backups run nightly to the `backups/` subdirectory of your data directory and are viewable in the Settings page.

---

## Remote Access

Use Tailscale rather than exposing port 8420 to the internet. The API uses HTTP (not HTTPS) — Tailscale encrypts traffic end-to-end.

Restrict access via Tailscale ACLs to only your own devices.

---

## Development

Requires Go 1.24+, Node 22+, and a C compiler (DuckDB uses CGO).

### Build

```bash
# Install: Go 1.24+, Node 22+, gcc/clang
cd frontend && npm ci && npm run build && cd ..
go build -tags release -o symbiont ./cmd/symbiont
```

### Running locally

```bash
cp .env.example .env
# edit .env

set -a && source .env && set +a
go run ./cmd/symbiont serve
```

Frontend dev server (live reload, proxies /api to :8421):

```bash
cd frontend && npm run dev
# Open http://localhost:5173
```

### Tests

```bash
go test ./...
```

---

## Documentation

| Document | Description |
|---|---|
| `CLAUDE.md` | Agent instructions — read before every coding session |
| `docs/symbiont-architecture.md` | Full technical architecture |
| `docs/impl-00-overview.md` | Implementation plan index |
| `docs/impl-01-*.md` through `impl-07-*.md` | Per-phase task lists |
| `docs/apex-api-notes.md` | Apex local API findings from DevTools capture |

---

## License

Private project. Not open source.
