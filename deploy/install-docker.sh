#!/usr/bin/env bash
set -euo pipefail

script_path="$(readlink -f -- "$0")"
repo_root="$(cd -- "$(dirname -- "$script_path")/.." && pwd)"
if [[ ${EUID} -ne 0 ]]; then
  echo "Запускаю установщик с sudo, чтобы безопасно настроить права папки данных."
  exec sudo -- bash "$script_path" "$@"
fi
cd "$repo_root"

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker не найден. Установи Docker Engine и Compose plugin, затем запусти этот скрипт снова." >&2
  exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "Не найден Docker Compose plugin (команда docker compose)." >&2
  exit 1
fi

env_file="$repo_root/.env"
data_dir="$repo_root/data"
game_file="$data_dir/config.env"
mkdir -p "$data_dir/logs"

if [[ ! -f "$env_file" ]]; then
  cp "$repo_root/.env.example" "$env_file"
  chmod 600 "$env_file"
fi
if [[ ! -f "$game_file" ]]; then
  printf 'TARGET_GAME=Hunt: Showdown 1896\n' > "$game_file"
  chmod 600 "$game_file"
fi

get_setting() {
  local file="$1" key="$2" line value
  while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ "$line" == "$key="* ]]; then
      value="${line#*=}"
      printf '%s' "${value%$'\r'}"
      return 0
    fi
  done < "$file"
  return 0
}

set_setting() {
  local file="$1" key="$2" value="$3" tmp="$file.tmp" line replaced=0
  umask 077
  : > "$tmp"
  while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ "$line" == "$key="* ]]; then
      printf '%s=%s\n' "$key" "$value" >> "$tmp"
      replaced=1
    else
      printf '%s\n' "$line" >> "$tmp"
    fi
  done < "$file"
  if (( replaced == 0 )); then printf '%s=%s\n' "$key" "$value" >> "$tmp"; fi
  chmod 600 "$tmp"
  mv -f -- "$tmp" "$file"
}

token="$(get_setting "$env_file" TELEGRAM_BOT_TOKEN)"
if [[ ! "$token" =~ ^[0-9]+:[A-Za-z0-9_-]{20,}$ || "$token" == 1234567890:AA* ]]; then
  read -r -s -p "Токен Telegram-бота (ввод скрыт): " token </dev/tty
  printf '\n'
  if [[ ! "$token" =~ ^[0-9]+:[A-Za-z0-9_-]{20,}$ ]]; then
    echo "Некорректный формат токена." >&2
    exit 1
  fi
  set_setting "$env_file" TELEGRAM_BOT_TOKEN "$token"
fi
unset token

admin_id="$(get_setting "$env_file" TELEGRAM_ADMIN_ID)"
if [[ ! "$admin_id" =~ ^[0-9]+$ || "$admin_id" == 123456789 ]]; then
  read -r -p "Ваш числовой Telegram ID: " admin_id </dev/tty
  if [[ ! "$admin_id" =~ ^[0-9]+$ ]]; then echo "Telegram ID должен содержать только цифры." >&2; exit 1; fi
  set_setting "$env_file" TELEGRAM_ADMIN_ID "$admin_id"
fi

game="$(get_setting "$game_file" TARGET_GAME)"
if [[ -z "$game" ]]; then
  read -r -p "Целевая игра: " game </dev/tty
  if [[ -z "$game" ]]; then echo "Название игры обязательно." >&2; exit 1; fi
  set_setting "$game_file" TARGET_GAME "$game"
else
  read -r -p "Текущая игра: $game. Оставить её? [Y/n]: " answer </dev/tty
  if [[ "$answer" =~ ^([nN]|no|нет|н)$ ]]; then
    read -r -p "Введите новую целевую игру: " game </dev/tty
    if [[ -z "$game" ]]; then echo "Название игры обязательно." >&2; exit 1; fi
    set_setting "$game_file" TARGET_GAME "$game"
  fi
fi

chown -R 10001:10001 "$data_dir"
find "$data_dir" -type d -exec chmod 750 {} +
chmod 600 "$game_file"
chmod 600 "$env_file"

docker compose up -d --build
docker compose ps

echo
echo "Готово. Отправь accounts.txt Telegram-боту документом."
echo "Логи контейнера: docker compose logs -f dropfarm"
