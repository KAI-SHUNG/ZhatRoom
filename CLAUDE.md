# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
# Build all binaries
go build -o bin/server cmd/server/main.go
go build -o bin/client cmd/client/main.go
go build -o bin/zhatroom cmd/zhatroom/main.go

# Local development (needs Docker PostgreSQL running first)
docker compose up -d
go run cmd/server/main.go          # terminal 1
go run cmd/client/main.go --usr alice  # terminal 2

# Run tests
go test ./...
go test ./internal/client/tui/ -run TestRenderMessages  # single test

# Production deploy (as root on server)
sudo ./scripts/deploy.sh
```

## Architecture

SSH-based chat room. Friends connect via `ssh chat@server`, SSH `command=` in authorized_keys forces the entrypoint script, which validates the user against PostgreSQL and execs the Bubble Tea TUI client. Client communicates with server over a Unix socket (`/tmp/zhatroom.sock`).

```
ssh chat@server
  → sshd matches Match User chat → reads /opt/zhatroom/authorized_keys
  → command="/opt/zhatroom/entrypoint.sh <uid> <username>"
  → entrypoint.sh validates uid in DB via docker exec psql
  → exec /opt/zhatroom/bin/client --id <uid> --usr <username> --socket /tmp/zhatroom.sock
  → client connects to Unix socket, sends "id,nickname\n" handshake
  → server Hub registers client, starts read loop
  → messages are newline-delimited JSON, broadcast to all connected clients
```

**Critical**: The `chat` system user's shell MUST be `/bin/sh`, not entrypoint.sh. SSH's `command=` executes via `$SHELL -c "<command>"` — if the shell is entrypoint.sh, it receives `-c` as `$1` and breaks.

## Key Components

- **cmd/server/main.go** — Unix socket listener, reads `id,nickname\n` handshake, delegates to Hub
- **cmd/client/main.go** — Parses flags (`--id`, `--usr`, `--socket`), creates Bubble Tea program with alt screen + mouse support
- **cmd/zhatroom/main.go** — Admin CLI: `zhatroom user add/list/remove`. `add` reads pubkey from stdin, generates random hex UID, writes to authorized_keys with `command=` restriction
- **internal/server/hub.go** — Central message hub with register/unregister/broadcast channels. Generates snowflake IDs for messages, persists to DB, broadcasts to all clients
- **internal/server/db.go** — GORM-based PostgreSQL storage (`Storage` struct). AutoMigrate creates tables on init
- **internal/client/tui/** — Bubble Tea TUI: Model (state), Update (events), View (render), Connector (socket I/O)
- **internal/protocol/protocol.go** — `Message` and `User` structs, JSON serialization. Message types: "chat", "command", "system"
- **scripts/entrypoint.sh** — SSH forced command. Validates user via `docker exec zhat_db psql`. Uses `exec` to replace shell with client binary
- **scripts/deploy.sh** — Full server setup: chat user, binaries, Docker PostgreSQL, systemd service, SSH config

## Protocol

Newline-delimited JSON over Unix socket. Handshake: client sends `id,nickname\n` on connect. Messages are `protocol.Message` structs with fields: `uid`, `type`, `from`, `from_id`, `ts`, `content`.

## Database

PostgreSQL 16 in Docker container `zhat_db` (user: `postgres`, password: `zhatroom`, db: `zhat_db`). GORM with AutoMigrate. Tables: `users` (id, nickname, created_at), `messages` (uid, type, from, from_id, ts, content).

## TUI Details

- Message alignment: own messages right-aligned, others left-aligned, system messages centered with `[SYSTEM]:` prefix
- Styles defined in `view.go`: `msgStyle` (white/15), `sysStyle` (green/43), `otherStyle` (cyan/14)
- Viewport height = terminal height - 3 (footer). Mouse wheel scrolling enabled via `tea.WithMouseAllMotion()`
- Input: textinput with 500 char limit, placeholder "输入消息..."
- Keys: Enter sends, Ctrl+C quits, PgUp/PgDn scrolls viewport

## SSH authorized_keys Format

```
command="/opt/zhatroom/entrypoint.sh <uid> <username>",no-port-forwarding,no-X11-forwarding,no-agent-forwarding ssh-ed25519 AAAA...
```

Do NOT use `restrict` — it includes `no-pty` which prevents the TUI from running.
