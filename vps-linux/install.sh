#!/usr/bin/env bash
set -euo pipefail

bundle_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
app_dir="/opt/tdm-light"
service_user="tdm-light"

if [[ ${EUID} -ne 0 ]]; then
  echo "Установщик настраивает systemd и /opt/tdm-light; запрашиваю sudo."
  exec sudo -- "$0" "$@"
fi

case "$(uname -m)" in
  x86_64|amd64) binary="tdm-light-linux-amd64" ;;
  aarch64|arm64) binary="tdm-light-linux-arm64" ;;
  *) echo "Неизвестная архитектура сервера: $(uname -m)" >&2; exit 1 ;;
esac

if [[ ! -f "$bundle_dir/$binary" ]]; then
  echo "В комплекте нет бинарного файла $binary." >&2
  exit 1
fi

if ! id "$service_user" >/dev/null 2>&1; then
  useradd --system --home-dir "$app_dir" --shell /usr/sbin/nologin "$service_user"
fi

install -d -o "$service_user" -g "$service_user" -m 0750 "$app_dir"
install -o "$service_user" -g "$service_user" -m 0755 "$bundle_dir/$binary" "$app_dir/tdm-light"

if [[ ! -f "$app_dir/config.json" ]]; then
  install -o "$service_user" -g "$service_user" -m 0600 "$bundle_dir/config.json.example" "$app_dir/config.json"
fi
if [[ ! -f "$app_dir/config.env" ]]; then
  install -o "$service_user" -g "$service_user" -m 0600 "$bundle_dir/config.env.example" "$app_dir/config.env"
fi
if [[ ! -e "$app_dir/accounts.txt" ]]; then
  install -o "$service_user" -g "$service_user" -m 0600 /dev/null "$app_dir/accounts.txt"
fi
install -d -o "$service_user" -g "$service_user" -m 0750 "$app_dir/logs"

config_file="$app_dir/config.env"
get_setting() {
  local key="$1" line
  while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ "$line" == "$key="* ]]; then
      local value="${line#*=}"
      printf '%s' "${value%$'\r'}"
      return 0
    fi
  done < "$config_file"
  return 0
}

set_setting() {
  local key="$1" value="$2" tmp="$config_file.tmp" line replaced=0
  umask 077
  : > "$tmp"
  while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ "$line" == "$key="* ]]; then
      printf '%s=%s\n' "$key" "$value" >> "$tmp"
      replaced=1
    else
      printf '%s\n' "$line" >> "$tmp"
    fi
  done < "$config_file"
  if (( replaced == 0 )); then
    printf '%s=%s\n' "$key" "$value" >> "$tmp"
  fi
  install -o "$service_user" -g "$service_user" -m 0600 "$tmp" "$config_file"
  rm -f -- "$tmp"
}

token="$(get_setting TELEGRAM_BOT_TOKEN)"
if [[ ! "$token" =~ ^[0-9]+:[A-Za-z0-9_-]{20,}$ || "$token" == 1234567890:AA* ]]; then
  echo "Для первого запуска настрой Telegram-бота. Токен вводится скрыто."
  read -r -s -p "Токен Telegram-бота: " token
  printf '\n'
  if [[ ! "$token" =~ ^[0-9]+:[A-Za-z0-9_-]{20,}$ ]]; then
    echo "Некорректный формат токена Telegram." >&2
    exit 1
  fi
  set_setting TELEGRAM_BOT_TOKEN "$token"
else
  echo "Существующий токен Telegram оставлен."
fi
unset token

admin_id="$(get_setting TELEGRAM_ADMIN_ID)"
if [[ ! "$admin_id" =~ ^[0-9]+$ || "$admin_id" == 123456789 ]]; then
  read -r -p "Ваш числовой Telegram ID: " admin_id
  if [[ ! "$admin_id" =~ ^[0-9]+$ ]]; then
    echo "Telegram ID должен содержать только цифры." >&2
    exit 1
  fi
  set_setting TELEGRAM_ADMIN_ID "$admin_id"
else
  echo "Существующий Telegram ID оставлен."
fi

game="$(get_setting TARGET_GAME)"
if [[ -z "$game" ]]; then
  read -r -p "Целевая игра (например, Hunt: Showdown 1896): " game
  if [[ -z "$game" ]]; then echo "Название игры обязательно." >&2; exit 1; fi
  set_setting TARGET_GAME "$game"
else
  read -r -p "Текущая игра: $game. Оставить её? [Y/n]: " answer
  if [[ "$answer" =~ ^([nN]|no|нет|н)$ ]]; then
    read -r -p "Введите новую целевую игру: " game
    if [[ -z "$game" ]]; then echo "Название игры обязательно." >&2; exit 1; fi
    set_setting TARGET_GAME "$game"
  fi
fi

chown "$service_user:$service_user" "$app_dir/config.json" "$app_dir/config.env" "$app_dir/accounts.txt"
chmod 0600 "$app_dir/config.json" "$app_dir/config.env" "$app_dir/accounts.txt"

install -o root -g root -m 0644 "$bundle_dir/tdm-light.service" /etc/systemd/system/tdm-light.service
systemctl daemon-reload
systemctl enable tdm-light.service
if systemctl is-active --quiet tdm-light.service; then
  systemctl restart tdm-light.service
else
  systemctl start tdm-light.service
fi

echo
echo "tdm light установлен и запущен как systemd-служба."
echo "Отправь Telegram-боту файл accounts.txt документом; он проверит его и перезагрузит майнер."
echo "Логи службы: journalctl -u tdm-light -f"
