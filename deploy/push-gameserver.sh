#!/usr/bin/env bash
# Run on your Mac after making a Linux Server build in Unity (game/Builds/Server/).
# Builds the game server image and loads it onto the home server over SSH.
#
# Usage: ./deploy/push-gameserver.sh you@your-server
set -euo pipefail
if [ $# -ne 1 ]; then echo "usage: $0 user@server"; exit 1; fi
cd "$(dirname "$0")/.."

docker build --platform linux/amd64 -t multiplayer-gameserver:latest -f game/server.Dockerfile game
docker save multiplayer-gameserver:latest | gzip | ssh "$1" 'gunzip | docker load'
echo "Loaded. On the server, set GAME_SERVER_IMAGE=multiplayer-gameserver:latest in deploy/.env, then: docker compose up -d"
