#!/usr/bin/env bash
set -euo pipefail

IMAGE_NAME="${1:-cry-112}"
DOCKER_PLATFORM="${2:-linux/amd64}"

docker build \
  --platform "$DOCKER_PLATFORM" \
  -f benzhi.Dockerfile \
  -t "$IMAGE_NAME" \
  .
