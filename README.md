# tdm light

**tdm light** is a Go based Twitch Drops watcher for multiple accounts. It can run on Windows or as a Docker container on an Ubuntu VPS. Account tokens and Telegram credentials are stored locally and must never be committed.

## Features

- Watches the configured game and selects live channels reported by Twitch with Drops enabled.
- Runs multiple accounts with staggered startup.
- Telegram bot commands for status, logs, changing the target game, restarting the worker, reward notifications, and uploading `accounts.txt`.
- Docker deployment with persistent account, configuration, and log files.

## Windows

1. Install Go 1.22 or newer.
2. Copy `accounts.txt.example` to `accounts.txt` and add accounts in the format expected by the parser.
3. Run `start.bat`. On first launch it creates local config files and prompts for the Telegram bot token, administrator Telegram ID, and game.
4. Send `accounts.txt` to the configured Telegram bot as a document to update the account list later.

`start.bat` builds and runs `tdm-light.exe`. Keep the console open while the worker runs.

## One-command Docker install on Ubuntu

Run this on a fresh Ubuntu VPS with `sudo` access:

```bash
curl -fsSL https://raw.githubusercontent.com/Xyloni/tdm-light/main/deploy/bootstrap-docker.sh | sudo bash
```

The installer installs Docker if needed, clones this repository into `/opt/tdm-light`, asks for the Telegram bot token, administrator Telegram ID, and target game, then builds and starts the container. Upload `accounts.txt` to the Telegram bot after it starts.

Useful commands:

```bash
cd /opt/tdm-light
sudo docker compose logs -f tdm-light
sudo docker compose restart tdm-light
sudo docker compose pull  # only if using a published image; normally update with git pull and --build
```

To update the source build:

```bash
cd /opt/tdm-light && sudo git pull && sudo docker compose up -d --build
```

The host directory `/opt/tdm-light/data` persists accounts, logs, and the selected game. Telegram credentials are in `/opt/tdm-light/.env`. Both locations are excluded from Git.

## Standalone Linux systemd bundle

To build a tarball with Linux binaries for `amd64` and `arm64`, run in PowerShell:

```powershell
.\build-vps.ps1
```

Copy `tdm-light-vps-ubuntu.tar.gz` to the Ubuntu server, extract it, enter the extracted directory, and run `sudo ./install.sh`. The installer configures the Telegram bot, administrator ID, and target game, then creates and starts `tdm-light.service`.

## Telegram bot

The bot accepts the configured administrator only. Available commands include `/status`, `/logs <login>`, `/game`, `/game <name>`, and `/restart`. Send `accounts.txt` as a document to replace the account list. The bot reports completed rewards when Twitch exposes that update.

## Configuration and privacy

- `config.env.example` and `config.json.example` are templates. The launch setup creates local configuration files.
- `accounts.txt.example` documents the expected file location; it contains no credentials.
- Never share or commit `accounts.txt`, `config.env`, `config.json`, `.env`, logs, or built binaries.
- On Docker, the target game is stored in `data/config.env`; Telegram secrets are read from `.env`.

## Development

```bash
go test ./...
go build ./...
```

---

# tdm light

**tdm light** — программа на Go для просмотра Twitch Drops с нескольких аккаунтов. Она работает в Windows или в Docker-контейнере на Ubuntu VPS. Токены аккаунтов и данные Telegram хранятся локально и не должны попадать в Git.

## Возможности

- Следит за выбранной игрой и выбирает живые каналы, которые Twitch помечает как Drops enabled.
- Подключает несколько аккаунтов с распределённым стартом.
- Команды Telegram-бота для статуса, логов, смены игры, перезапуска, уведомлений о наградах и загрузки `accounts.txt`.
- Docker-развёртывание с постоянным хранением аккаунтов, настроек и логов.

## Windows

1. Установи Go версии 1.22 или новее.
2. Скопируй `accounts.txt.example` в `accounts.txt` и добавь аккаунты в формате, который понимает парсер.
3. Запусти `start.bat`. При первом запуске он создаст локальные файлы настроек и запросит токен Telegram-бота, Telegram ID администратора и игру.
4. Чтобы обновить список аккаунтов позже, отправь `accounts.txt` настроенному боту документом.

`start.bat` собирает и запускает `tdm-light.exe`. Пока майнер работает, не закрывай консоль.

## Установка Docker на Ubuntu одной командой

Запусти на чистом Ubuntu VPS с доступом `sudo`:

```bash
curl -fsSL https://raw.githubusercontent.com/Xyloni/tdm-light/main/deploy/bootstrap-docker.sh | sudo bash
```

Установщик при необходимости установит Docker, скачает репозиторий в `/opt/tdm-light`, запросит токен Telegram-бота, Telegram ID администратора и игру, затем соберёт и запустит контейнер. После запуска отправь боту файл `accounts.txt`.

Полезные команды:

```bash
cd /opt/tdm-light
sudo docker compose logs -f tdm-light
sudo docker compose restart tdm-light
```

Обновление исходников и пересборка:

```bash
cd /opt/tdm-light && sudo git pull && sudo docker compose up -d --build
```

Каталог `/opt/tdm-light/data` хранит аккаунты, логи и выбранную игру. Секреты Telegram находятся в `/opt/tdm-light/.env`. Эти данные исключены из Git.

## Отдельная Linux-версия с systemd

Чтобы собрать архив с Linux-бинарниками для `amd64` и `arm64`, выполни в PowerShell:

```powershell
.\build-vps.ps1
```

Скопируй `tdm-light-vps-ubuntu.tar.gz` на Ubuntu-сервер, распакуй архив, перейди в каталог с файлами и запусти `sudo ./install.sh`. Установщик настроит Telegram-бота, ID администратора и игру, затем создаст и запустит службу `tdm-light.service`.

## Telegram-бот

Бот принимает команды только от заданного администратора. Доступны `/status`, `/logs <логин>`, `/game`, `/game <название>` и `/restart`. Чтобы заменить список аккаунтов, отправь `accounts.txt` документом. Бот сообщает о получении наград, когда Twitch передаёт такое обновление.

## Настройки и приватность

- `config.env.example` и `config.json.example` — шаблоны; локальные файлы настроек создаются при запуске.
- `accounts.txt.example` показывает расположение списка, но не содержит учётных данных.
- Не публикуй `accounts.txt`, `config.env`, `config.json`, `.env`, логи и собранные бинарные файлы.
- В Docker целевая игра хранится в `data/config.env`, а секреты Telegram читаются из `.env`.

## Разработка

```bash
go test ./...
go build ./...
```
