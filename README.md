# MultiplayerHomeServer

Cross-platform multiplayer game (PC, Mac, Linux, Android, iOS) with a self-hosted backend running on a home Ubuntu PC.

**Done when:** a friend on their phone and you on Linux join the same lobby code from different houses and play.

## How it works

```
Players (PC / phone / Linux)
        │  1. log in, host or join a lobby (HTTP :8080)
        │  2. play on the game server the API gives back (UDP 7770–7790)
Tailscale  or  router port forward
        │
┌──────────── Home server (Ubuntu + Docker Compose) ─────────────┐
│  Go API (:8080)   login, lobby codes, starts/stops game servers │
│  Game servers     one Unity container per lobby, UDP            │
│  Postgres         accounts                                      │
│  Redis            lobbies, port reservations                    │
│  Prometheus + Grafana (:3000)   players, servers, CPU, memory   │
└────────────────────────────────────────────────────────────────┘
```

When someone hosts a lobby, the API reserves a UDP port, starts a game server container on it, and returns the code. Game servers send a heartbeat every 5 seconds with their player count. The API shuts a game server down when its lobby expires (2 hours), it stops sending heartbeats (2 minutes), or it has had no players for 5 minutes.

## Stack

| Part | Tech |
|---|---|
| Game client + game server | Unity 6 (C#) + FishNet, headless Linux build for the server |
| Backend API | Go (net/http), JWT auth, Docker Engine API for game servers |
| Database | PostgreSQL |
| Cache / live state | Redis |
| Running it | Docker + Docker Compose on Ubuntu Server |
| Remote access | Tailscale first, port forward + DDNS later |
| CI/CD | GitHub Actions build check, `update.sh` auto-deploys from GitHub |
| Monitoring | Prometheus + node-exporter + Grafana |
| Backups | nightly `pg_dump` via cron |

## Repo layout

```
api/            Go backend
game/           Unity project (scripts in Assets/Scripts, setup steps in game/README.md)
deploy/         docker-compose.yml, .env, monitoring config, backup/update/push scripts
server-setup/   one-time Ubuntu server setup script
.github/        GitHub Actions workflow (builds the API on every push)
```

## Status

| Phase | Status |
|---|---|
| 1. Server setup | Script ready: `server-setup/setup-ubuntu.sh` |
| 2. Unity game | Scripts written; scene + prefab are made in the editor (`game/README.md`) |
| 3. Go backend | Done |
| 4. Game server per lobby | Done (start, heartbeat, auto-shutdown) |
| 5. Friends | Steps below |
| 6. Automation + monitoring | Done: CI, auto-update, Grafana dashboard, backups |

## Setup

### 1. Server (once)
On the Ubuntu PC:
```bash
git clone https://github.com/twoSquaredHoon/MultiplayerHomeServer.git
cd MultiplayerHomeServer
bash server-setup/setup-ubuntu.sh
```
Then in the router admin page, give the server a fixed LAN IP (DHCP reservation).

### 2. Backend
```bash
cd deploy
cp .env.example .env     # set passwords, PUBLIC_HOST = output of: tailscale ip -4
docker compose up -d --build
curl localhost:8080/health
```

### 3. Game
Follow `game/README.md`, then load the server build onto the home server from your Mac:
```bash
./deploy/push-gameserver.sh you@your-server
```
On the server, set `GAME_SERVER_IMAGE=multiplayer-gameserver:latest` in `deploy/.env` and run `docker compose up -d`.

### 4. Let friends in (Phase 5)
**Tailscale (start here):**
1. tailscale.com admin console → Machines → your server → **Share** → send the link to a friend.
2. Friend installs Tailscale (Windows, Mac, Linux, iOS, Android) and accepts the share.
3. Clients use `http://<server Tailscale IP>:8080` as the API URL, the same as `PUBLIC_HOST`.

**Port forwarding (later, anyone on the internet):**
1. Router admin (ASUS) → WAN → Virtual Server / Port Forwarding: TCP **8080** and UDP **7770–7790** → the server's LAN IP.
2. ASUS → WAN → DDNS → turn on, pick a name like `yourname.asuscomm.com`.
3. Set `PUBLIC_HOST=yourname.asuscomm.com` in `deploy/.env`, set the clients' API URL to `http://yourname.asuscomm.com:8080`, and `docker compose up -d`.

Don't forward port 3000 (Grafana); open it over Tailscale only.

### 5. Automation (Phase 6)
On the server, `crontab -e` and add:
```
*/10 * * * * /home/YOU/MultiplayerHomeServer/deploy/update.sh >> /tmp/update.log 2>&1
0 4 * * *    /home/YOU/MultiplayerHomeServer/deploy/backup.sh >> /tmp/backup.log 2>&1
```
- `update.sh` pulls from GitHub every 10 minutes and rebuilds if something changed.
- `backup.sh` saves the database to `deploy/backups/` every night and keeps 14 days.
- Restore a backup: `gunzip -c backups/game-DATE.sql.gz | docker compose exec -T postgres psql -U game game`


### 6. Monitoring
Open `http://<server Tailscale IP>:3000`, log in as `admin` / `GRAFANA_PASSWORD`, open the **Multiplayer server** dashboard: players online, game servers, lobbies, accounts, CPU and memory.

## API

| Method | Path | Auth | Returns |
|---|---|---|---|
| GET | `/health` | no | `ok` |
| GET | `/metrics` | no | Prometheus metrics |
| POST | `/register` | no | `{id}` |
| POST | `/login` | no | `{token}` |
| POST | `/lobbies` | token | `{code, host, port}`, starts a game server |
| POST | `/lobbies/{code}/join` | token | `{code, host, port}` |
| POST | `/servers/heartbeat` | `X-Server-Key` | game servers only: `{port, players}` |

Test without the game:
```bash
curl -X POST localhost:8080/register -d '{"username":"hoon","password":"pass1234"}'
TOKEN=$(curl -s -X POST localhost:8080/login -d '{"username":"hoon","password":"pass1234"}' | sed 's/.*"token":"\([^"]*\)".*/\1/')
curl -X POST localhost:8080/lobbies -H "Authorization: Bearer $TOKEN"
curl -X POST localhost:8080/lobbies/ABCD/join -H "Authorization: Bearer $TOKEN"
```

## Resume line

Self-hosted cross-platform multiplayer backend: Go API that spins up per-match Dockerized Unity game servers, Postgres/Redis, CI/CD, Prometheus/Grafana monitoring.
