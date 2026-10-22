#!/usr/bin/env bash
# Pull the latest code from GitHub and rebuild anything that changed.
# Cron (crontab -e):  */10 * * * * /home/YOU/MultiplayerHomeServer/deploy/update.sh >> /tmp/update.log 2>&1
set -euo pipefail
cd "$(dirname "$0")/.."
before=$(git rev-parse HEAD)
git pull --ff-only
after=$(git rev-parse HEAD)
if [ "$before" != "$after" ]; then
  echo "$(date): updating $before -> $after"
  cd deploy && docker compose up -d --build
fi
