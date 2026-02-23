#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2025 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail

: "${TEMPORAL_CLI_ADDRESS:=localhost:7233}"

retry() {
  local tries="${1:-30}"
  local delay="${2:-1}"
  shift 2 || true
  local n=1
  while true; do
    if "$@"; then
      return 0
    fi
    if [ "$n" -ge "$tries" ]; then
      return 1
    fi
    n=$((n + 1))
    sleep "$delay"
  done
}

ensure_attr() {
  local name="$1"
  local type="$2"

  if temporal operator search-attribute list | grep -q "\b${name}\b"; then
    echo "Temporal search attribute already exists: ${name}"
    return 0
  fi

  temporal operator search-attribute create --name "${name}" --type "${type}"
}

retry 60 1 ensure_attr "PipelineIdentifier" "Keyword"
