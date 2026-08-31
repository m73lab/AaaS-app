#!/usr/bin/env bash
# Renders config.homelab02.yaml from the template, injecting the Redis password
# (the master key is supplied to the proxy via container env, not the config).
# Output is gitignored; never commit the rendered file.
set -euo pipefail

REDIS_PASSWORD="${REDIS_PASSWORD:-change-me-strong}"

sed -e "s/__REDIS_PASSWORD__/$REDIS_PASSWORD/" \
    config.homelab02.yaml.template > config.homelab02.yaml

echo "rendered config.homelab02.yaml (gitignored)"
