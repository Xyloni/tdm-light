#!/usr/bin/env bash
set -euo pipefail

if (( EUID != 0 )); then exec sudo -- bash "$0" "$@"; fi
if ! docker compose version >/dev/null 2>&1; then
  echo "Install Docker Engine and its Compose plugin before running this installer." >&2
  exit 1
fi
destination="${TDMLIGHT_DIR:-/opt/tdm-light-vps}"
if [[ "$destination" != /* || "$destination" == "/" || "$destination" == *$'\n'* ]]; then
  echo "TDMLIGHT_DIR must be an absolute installation directory." >&2
  exit 1
fi
if [[ -d "$destination" && ! -f "$destination/compose.yaml" && -n "$(find "$destination" -mindepth 1 -maxdepth 1 -print -quit)" ]]; then
  echo "Directory contains another installation. Choose a new TDMLIGHT_DIR." >&2
  exit 1
fi
staging="$(mktemp -d)"
trap 'rm -rf -- "$staging"' EXIT
curl --fail --location --proto '=https' --tlsv1.2 --retry 3 \
  https://codeload.github.com/Xyloni/tdm-light/tar.gz/refs/heads/main -o "$staging/source.tar.gz"
tar -xzf "$staging/source.tar.gz" -C "$staging" --strip-components=2 \
  tdm-light-main/vps/compose.yaml tdm-light-main/vps/.env.example \
  tdm-light-main/vps/install.sh tdm-light-main/vps/update.sh tdm-light-main/vps/README.md
install -d -m 750 "$destination"
for name in compose.yaml .env.example README.md; do install -m 644 "$staging/$name" "$destination/$name"; done
for name in install.sh update.sh; do install -m 755 "$staging/$name" "$destination/$name"; done
bash "$destination/install.sh"
