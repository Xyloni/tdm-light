<div align="center">

# 🎮 TwitchDrops Miner Light

### A lightweight, multi-account Twitch Drops watcher with Telegram control

[English](#english) · [Русский](#русский)

🟣 Twitch Drops &nbsp;·&nbsp; 🤖 Telegram bot &nbsp;·&nbsp; 🐳 Docker &nbsp;·&nbsp; 🪟 Windows &nbsp;·&nbsp; 🐧 Ubuntu VPS

</div>

---

## English

TwitchDrops Miner Light is a Go application that watches live channels for a selected Twitch Drops campaign across multiple accounts. Run it on Windows or deploy it to an Ubuntu VPS with Docker Compose.

### ✨ Features

- 🎯 Looks for live channels with Drops enabled for the selected game.
- 👥 Runs multiple accounts with staggered startup.
- 🤖 Telegram controls for status, logs, game selection, restart, reward notifications, and account file uploads.
- 💾 Keeps account data, logs, and game settings in persistent local storage.
- 🐳 Builds and runs with Docker Compose on a VPS.

### 🚀 Quick start on Ubuntu VPS

On Ubuntu with Docker Engine, the Docker Compose plugin, and `sudo` access, run:

```bash
curl -fsSL https://raw.githubusercontent.com/Xyloni/tdm-light/main/deploy/bootstrap-docker.sh | sudo bash
```

The bootstrap script downloads the project to `/opt/tdm-light`, asks for your Telegram bot token, Telegram administrator ID, and target game, then builds and starts the container. Docker must already be installed; the script stops with an install hint if Docker or Compose is missing. Upload `accounts.txt` to the bot after startup.

> Updates currently use a source build on the VPS. Apply them with `git pull` and rebuild the container:

```bash
cd /opt/tdm-light
sudo git pull
sudo docker compose up -d --build
```

Useful commands:

| Action | Command |
| --- | --- |
| Follow logs | `cd /opt/tdm-light && sudo docker compose logs -f tdm-light` |
| Restart | `cd /opt/tdm-light && sudo docker compose restart tdm-light` |
| Check status | `cd /opt/tdm-light && sudo docker compose ps` |

### 🪟 Run on Windows

1. Install Go 1.22 or newer.
2. Run `start.bat`. On first launch, it creates local configuration files and prompts for the Telegram bot token, administrator ID, and game. It then creates a sample `accounts.txt` and exits so you can fill it in.
3. Add accounts to `accounts.txt` using the format below, then run `start.bat` again.
4. Keep the console open while the miner runs. The launcher builds and starts `tdm-light.exe`.

To replace the account list later, send `accounts.txt` to the configured Telegram bot as a document.

### 👥 Account file format

Each account goes on its own line, with six colon-separated fields:

```text
email:login:password:oauth_token:user_ID:device_ID
```

The OAuth value is the token without the `oauth:` prefix. The parser also accepts a group marker such as `1:` on its own line; accounts after it belong to that group until the next marker. Keep this file private because it contains account credentials.

### 🐧 Standalone Linux systemd bundle

Build Linux binaries for `amd64` and `arm64` from PowerShell:

```powershell
.\build-vps.ps1
```

Copy `tdm-light-vps-ubuntu.tar.gz` to the server, extract it, and run `sudo ./install.sh`. The installer configures Telegram and the target game, then creates and starts `tdm-light.service`.

### 🤖 Telegram bot

The bot accepts commands from the configured administrator. Available commands include:

| Command | Action |
| --- | --- |
| `/status` | Show miner status |
| `/logs <login>` | Show logs for an account |
| `/game` | Show or choose the target game |
| `/game <name>` | Change the target game |
| `/restart` | Restart the miner |

Send `accounts.txt` as a document to update the account list. Reward notifications are sent when Twitch reports a reward update.

### 🔐 Configuration and privacy

- `config.env.example` and `config.json.example` are configuration templates.
- Docker stores persistent data in `/opt/tdm-light/data` and Telegram secrets in `/opt/tdm-light/.env`.
- Never commit or share `accounts.txt`, `config.env`, `config.json`, `.env`, logs, or built binaries.
- Local account and configuration files are excluded from Git.

### 🧰 Development

```bash
go test ./...
go build ./...
```

---

## Русский

TwitchDrops Miner Light — приложение на Go для просмотра Twitch-стримов с активными Drops по выбранной игре сразу с нескольких аккаунтов. Его можно запускать в Windows или развернуть на Ubuntu VPS через Docker Compose.

### ✨ Возможности

- 🎯 Ищет живые каналы с включёнными Drops для выбранной игры.
- 👥 Подключает несколько аккаунтов с распределённым стартом.
- 🤖 Управление через Telegram: статус, логи, выбор игры, перезапуск, уведомления о наградах и загрузка списка аккаунтов.
- 💾 Хранит аккаунты, логи и настройки игры в постоянном локальном каталоге.
- 🐳 Собирается и запускается на VPS через Docker Compose.

### 🚀 Быстрая установка на Ubuntu VPS

На Ubuntu VPS с установленными Docker Engine, плагином Docker Compose и доступом `sudo` выполни:

```bash
curl -fsSL https://raw.githubusercontent.com/Xyloni/tdm-light/main/deploy/bootstrap-docker.sh | sudo bash
```

Скрипт скачает проект в `/opt/tdm-light`, запросит токен Telegram-бота, ID администратора и целевую игру, затем соберёт и запустит контейнер. Docker должен быть установлен заранее; если Docker или Compose не найдены, скрипт завершится и попросит установить их. После запуска отправь боту `accounts.txt` документом.

> Сейчас обновление выполняется сборкой исходников на самом VPS. Чтобы установить обновления, выполни `git pull` и пересобери контейнер:

```bash
cd /opt/tdm-light
sudo git pull
sudo docker compose up -d --build
```

Полезные команды:

| Действие | Команда |
| --- | --- |
| Смотреть логи | `cd /opt/tdm-light && sudo docker compose logs -f tdm-light` |
| Перезапустить | `cd /opt/tdm-light && sudo docker compose restart tdm-light` |
| Проверить статус | `cd /opt/tdm-light && sudo docker compose ps` |

### 🪟 Запуск в Windows

1. Установи Go версии 1.22 или новее.
2. Запусти `start.bat`. При первом запуске он создаст локальные настройки, запросит токен Telegram-бота, ID администратора и игру, затем создаст образец `accounts.txt` и завершится, чтобы ты заполнил файл.
3. Добавь аккаунты в `accounts.txt` в формате ниже и запусти `start.bat` ещё раз.
4. Оставь консоль открытой, пока работает майнер. Батник соберёт и запустит `tdm-light.exe`.

Чтобы позже заменить список аккаунтов, отправь `accounts.txt` настроенному Telegram-боту документом.

### 👥 Формат файла аккаунтов

Каждый аккаунт указывается на отдельной строке в виде шести полей, разделённых двоеточиями:

```text
email:login:password:oauth_token:user_ID:device_ID
```

OAuth-токен указывается без префикса `oauth:`. Парсер поддерживает маркеры групп, например `1:` на отдельной строке. Следующие аккаунты относятся к этой группе до следующего маркера. Храни файл в секрете: в нём есть данные аккаунтов.

### 🐧 Отдельный Linux-комплект с systemd

Собери Linux-бинарники для `amd64` и `arm64` из PowerShell:

```powershell
.\build-vps.ps1
```

Скопируй `tdm-light-vps-ubuntu.tar.gz` на сервер, распакуй архив и запусти `sudo ./install.sh`. Установщик настроит Telegram и целевую игру, затем создаст и запустит службу `tdm-light.service`.

### 🤖 Telegram-бот

Бот принимает команды от заданного администратора. Доступны:

| Команда | Действие |
| --- | --- |
| `/status` | Показать состояние майнера |
| `/logs <логин>` | Показать логи аккаунта |
| `/game` | Показать или выбрать целевую игру |
| `/game <название>` | Сменить целевую игру |
| `/restart` | Перезапустить майнер |

Чтобы обновить список аккаунтов, отправь `accounts.txt` документом. Уведомления о наградах приходят, когда Twitch сообщает об их обновлении.

### 🔐 Настройки и приватность

- `config.env.example` и `config.json.example` — шаблоны настроек.
- Docker хранит постоянные данные в `/opt/tdm-light/data`, а секреты Telegram — в `/opt/tdm-light/.env`.
- Не публикуй и не отправляй посторонним `accounts.txt`, `config.env`, `config.json`, `.env`, логи и собранные бинарники.
- Локальные файлы аккаунтов и настроек исключены из Git.

### 🧰 Разработка

```bash
go test ./...
go build ./...
```
