#!/usr/bin/env bash
# 部署脚本：在 10.0.7.225 上拉取最新镜像并重启开发环境

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"

REPO="ghcr.io/bernylinville/inspection-tool"
COMPOSE_FILE="docker-compose-dev.yml"

cd "${REPO_ROOT}"

echo "=== Pulling latest images ==="
docker pull "${REPO}/backend:dev"
docker pull "${REPO}/frontend:dev"

echo "=== Restarting containers ==="
docker compose -f "${COMPOSE_FILE}" down
docker compose -f "${COMPOSE_FILE}" up -d

echo "=== Service status ==="
docker compose -f "${COMPOSE_FILE}" ps
