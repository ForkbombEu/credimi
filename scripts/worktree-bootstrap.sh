#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2025 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLASSIC=0

usage() {
	cat <<'USAGE'
Usage: scripts/worktree-bootstrap.sh [--classic]

For parallel worktrees, requires Worktrunk (`wt`). Copies allowlisted
gitignored paths via `wt step copy-ignored --require-include`, writes
.env.worktree with unique ports, syncs webapp/.env URLs, initializes
submodules, and ensures .bin tools exist.

On the primary checkout (or with --classic), only classic ports are written;
Worktrunk is not required.
USAGE
}

while [[ $# -gt 0 ]]; do
	case "$1" in
	--classic)
		CLASSIC=1
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		echo "unknown arg: $1" >&2
		usage >&2
		exit 1
		;;
	esac
done

cd "${ROOT_DIR}"

primary="$(git worktree list --porcelain | awk '/^worktree / { print $2; exit }')"
if [[ -n "${primary}" && "$(cd "${primary}" && pwd -P)" == "$(pwd -P)" ]]; then
	echo "bootstrap on primary worktree: classic ports only"
	CLASSIC=1
fi

if [[ "${CLASSIC}" -eq 0 ]]; then
	if ! command -v wt >/dev/null 2>&1; then
		echo "error: Worktrunk (\`wt\`) is required for parallel worktrees." >&2
		echo "Install: brew install worktrunk && wt config shell install" >&2
		echo "docs: https://worktrunk.dev/" >&2
		exit 1
	fi
	echo "copying allowlisted ignored files via Worktrunk"
	wt step copy-ignored --require-include
	./scripts/worktree-env.sh write
else
	./scripts/worktree-env.sh write --classic
fi

./scripts/worktree-env.sh sync-urls

git submodule update --init --recursive

if [[ ! -x .bin/stepci-captured-runner || ! -x .bin/et-tu-cesr ]]; then
	echo "running make tools"
	make tools
fi

echo
echo "bootstrap complete."
echo "  cd ${ROOT_DIR} && make dev"
./scripts/worktree-env.sh print
