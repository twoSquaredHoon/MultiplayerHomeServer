#!/usr/bin/env bash
# Phase 1: one-time setup for the home server (Ubuntu Server 24.04).
# Run on the server:  bash setup-ubuntu.sh
set -euo pipefail

echo "== Updating packages =="
sudo apt update && sudo apt upgrade -y
sudo apt install -y git curl ufw openssh-server

echo "== Firewall =="
sudo ufw allow OpenSSH
sudo ufw allow 8080/tcp          # Go API
sudo ufw allow 7770:7790/udp     # game servers
sudo ufw --force enable

echo "== Docker =="
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker "$USER"

echo "== Tailscale =="
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up

echo
echo "Done. Log out and back in so docker works without sudo."
echo "Tailscale IP: $(tailscale ip -4 2>/dev/null || echo 'run: tailscale ip -4')"
echo "Next: set a DHCP reservation for this PC in the router, then copy your SSH key with ssh-copy-id."
