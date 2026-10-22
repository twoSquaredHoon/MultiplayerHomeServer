#!/usr/bin/env bash
# Nightly Postgres backup. Keeps the last 14 days in deploy/backups/.
# Cron (crontab -e):  0 4 * * * /home/YOU/MultiplayerHomeServer/deploy/backup.sh
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p backups
docker compose exec -T postgres pg_dump -U game game | gzip > "backups/game-$(date +%F).sql.gz"
find backups -name 'game-*.sql.gz' -mtime +14 -delete
echo "backup done: backups/game-$(date +%F).sql.gz"
