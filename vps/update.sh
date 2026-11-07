#!/usr/bin/env bash
set -euo pipefail
cd -- "$(dirname -- "$(readlink -f -- "$0")")"
if (( EUID != 0 )); then exec sudo -- bash "$PWD/update.sh" "$@"; fi

compose() { docker compose -f compose.yaml "$@"; }
set_image() {
  local value="$1" tmp line
  tmp="$(mktemp .env.XXXXXX)"
  chmod 600 "$tmp"
  while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ "$line" != TDMLIGHT_IMAGE=* ]]; then printf '%s\n' "$line" >> "$tmp"; fi
  done < .env
  printf 'TDMLIGHT_IMAGE=%s\n' "$value" >> "$tmp"
  mv -f -- "$tmp" .env
}

if [[ ! -f .env ]]; then echo "Run install.sh first." >&2; exit 1; fi
if [[ "${1:-}" == "--rollback" ]]; then
  if [[ ! -s .previous-image ]]; then echo "No previous image saved." >&2; exit 1; fi
  candidate="$(cat .previous-image)"
else
  candidate="${1:-$(sed -n 's/^TDMLIGHT_IMAGE=//p' .env | tail -n 1)}"
fi
if [[ ! "$candidate" =~ ^ghcr\.io/xyloni/tdm-light(:[a-zA-Z0-9_.-]+|@sha256:[a-f0-9]{64})$ ]]; then
  echo "Use a ghcr.io/xyloni/tdm-light tag or digest." >&2
  exit 1
fi
container="$(compose ps -a -q tdm-light)"
previous=""
previous_id=""
if [[ -n "$container" ]]; then
  previous_id="$(docker inspect --format '{{.Image}}' "$container")"
  previous="$(docker image inspect --format '{{index .RepoDigests 0}}' "$previous_id" 2>/dev/null || true)"
fi
# Pull before stopping the running miner.
TDMLIGHT_IMAGE="$candidate" compose pull tdm-light
next_id="$(docker image inspect --format '{{.Id}}' "$candidate")"
if TDMLIGHT_IMAGE="$candidate" compose up -d --no-deps --no-build --force-recreate tdm-light; then
  sleep 2
  container="$(compose ps -a -q tdm-light)"
  state="$(docker inspect --format '{{.State.Status}} {{.State.ExitCode}}' "$container")"
  if [[ "$state" != running* && "$state" != "exited 0" ]]; then
    echo "New container failed to start." >&2
  else
    if [[ "$previous" == ghcr.io/xyloni/tdm-light@sha256:* && "$previous_id" != "$next_id" ]]; then
      umask 077
      printf '%s\n' "$previous" > .previous-image
    fi
    set_image "$candidate"
    echo "Updated. Account data and bot settings are in ./data."
    exit 0
  fi
fi
if [[ "$previous" == ghcr.io/xyloni/tdm-light@sha256:* ]]; then
  echo "Restoring the previous image..." >&2
  TDMLIGHT_IMAGE="$previous" compose up -d --no-deps --no-build --pull never --force-recreate tdm-light
  set_image "$previous"
fi
exit 1
