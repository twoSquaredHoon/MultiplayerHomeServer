# MultiplayerHomeServer

Cross-platform multiplayer game (PC, Mac, Linux, Android, iOS) with a self-hosted backend running on a home Ubuntu PC.

**Done when:** a friend on their phone and you on Linux join the same lobby code from different houses and play.

## Architecture

```
Players (PC / phone / Linux)
        │
Tailscale  or  router port forward
        │
┌─────── Home server (Ubuntu + Docker Compose) ───────┐
│  Go API  (:8080)  ── login, lobby codes             │
│  Game servers (UDP 7770–7790) ── one per match      │
│  Postgres ── accounts, saves                        │
│  Redis    ── lobbies, online players                │
└─────────────────────────────────────────────────────┘
```

## Stack

| Part | Tech |
|---|---|
| Game client + game server | Unity (C#) + FishNet, headless Linux build for the server |
| Backend API | Go (net/http), JWT auth |
| Database | PostgreSQL |
| Cache / live state | Redis |
| Running it | Docker + Docker Compose on Ubuntu Server |
| Remote access | Tailscale first, port forward + DDNS later |
| CI/CD | GitHub Actions → GHCR |
| Monitoring | Prometheus + Grafana |

## Repo layout

```
api/            Go backend (login, lobbies)
game/           Unity project (client + headless server build)
deploy/         docker-compose.yml, .env
server-setup/   one-time Ubuntu server setup script
.github/        CI workflows
```

## Phases

### Phase 1 — Server setup (weekend)
- [ ] Install Ubuntu Server on the spare PC
- [ ] SSH keys, `ufw` firewall
- [ ] Fixed LAN IP (DHCP reservation in router)
- [ ] Docker, Docker Compose, Tailscale → run `server-setup/setup-ubuntu.sh`

### Phase 2 — Minimal game (1–2 weeks)
- [ ] Unity + FishNet: players move and see each other
- [ ] Headless Linux server build + `Dockerfile`
- [ ] Two clients on home Wi-Fi

### Phase 3 — Go backend (1–2 weeks)
- [x] Starter API in `api/` (register, login, create lobby, join lobby)
- [ ] Run with `docker compose up` and test with curl
- [ ] Unity client calls the API, then connects to the returned IP/port

### Phase 4 — Match servers (1 week)
- [ ] API starts a game server container per lobby (Docker SDK for Go)
- [ ] Game server heartbeat → API; API shuts down empty servers

### Phase 5 — Let friends in (few days)
- [ ] Tailscale: share server with friends
- [ ] Later: port forward TCP 8080 + UDP 7770–7790, ASUS DDNS

### Phase 6 — Automation + monitoring (1 week)
- [ ] GitHub Actions builds images → GHCR, server pulls new versions
- [ ] Prometheus + Grafana (players online, matches, CPU)
- [ ] Nightly `pg_dump` backup via cron

## Run the backend locally

```bash
cd deploy
cp .env.example .env      # then change JWT_SECRET and POSTGRES_PASSWORD
docker compose up --build
```

Test it:

```bash
curl -X POST localhost:8080/register -d '{"username":"hoon","password":"pass1234"}'
TOKEN=$(curl -s -X POST localhost:8080/login -d '{"username":"hoon","password":"pass1234"}' | sed 's/.*"token":"\([^"]*\)".*/\1/')
curl -X POST localhost:8080/lobbies -H "Authorization: Bearer $TOKEN"
curl -X POST localhost:8080/lobbies/ABCD/join -H "Authorization: Bearer $TOKEN"
```

## API

| Method | Path | Auth | Returns |
|---|---|---|---|
| GET | `/health` | no | `ok` |
| POST | `/register` | no | `{id}` |
| POST | `/login` | no | `{token}` |
| POST | `/lobbies` | yes | `{code, host, port}` |
| POST | `/lobbies/{code}/join` | yes | `{code, host, port}` |

## Resume line

Self-hosted cross-platform multiplayer backend: Go API, per-match Dockerized game servers, Postgres/Redis, CI/CD, Grafana monitoring.
