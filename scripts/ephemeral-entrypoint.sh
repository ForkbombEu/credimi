#!/usr/bin/env bash
set -euo pipefail

seed_db_path="${SEED_DB_PATH:-/seed/pb_data/data.db}"
data_db_path="${DATA_DB_PATH:-/app/pb_data/data.db}"

if [[ -f "${seed_db_path}" ]]; then
  mkdir -p "$(dirname "${data_db_path}")"
  cp "${seed_db_path}" "${data_db_path}"

  if command -v pinkmask >/dev/null 2>&1; then
    masked_tmp="${data_db_path}.masked"
    if [[ -n "${PINKMASK_ARGS:-}" ]]; then
      pinkmask ${PINKMASK_ARGS}
    else
      pinkmask -i "${data_db_path}" -o "${masked_tmp}"
      mv "${masked_tmp}" "${data_db_path}"
    fi
  else
    echo "pinkmask not found; skipping masking step" >&2
  fi
fi

exec "$@"
