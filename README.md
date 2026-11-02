<div align="center">

# 🎮 TwitchDrops Miner Light

### A lightweight, multi-account Twitch Drops watcher with Telegram or Discord control

[English](#english) · [Русский](#русский)

🟣 Twitch Drops &nbsp;·&nbsp; 🤖 Telegram / Discord &nbsp;·&nbsp; 🐳 Docker &nbsp;·&nbsp; 🪟 Windows &nbsp;·&nbsp; 🐧 Ubuntu VPS

</div>

---

## English

TwitchDrops Miner Light is a Go application that watches live channels for a selected Twitch Drops campaign across multiple accounts. Run it on Windows or deploy it to an Ubuntu VPS with Docker Compose.

### ✨ Features

- 🎯 Looks for live channels with Drops enabled for the selected game.
- 👥 Runs multiple accounts with staggered startup.
- ⏱ Tracks confirmed Inventory minutes; rewards are claimed manually.
- 📈 Shows watch-time crediting speed in Telegram or Discord and recovers persistently slow viewing.
- 🤖 Telegram or Discord controls (choose on first launch) for status, logs, game selection, restart, reward notifications, and account file uploads.
- 💾 Keeps account data, logs, and game settings in persistent local storage.
- 🐳 Builds and runs with Docker Compose on a VPS.

### 🚀 Quick start on Ubuntu VPS

On Ubuntu with Docker Engine, the Docker Compose plugin, and `sudo` access, run:

```bash
curl -fsSL https://raw.githubusercontent.com/Xyloni/tdm-light/main/deploy/bootstrap-docker.sh | sudo bash
```

The bootstrap script downloads the project to `/opt/tdm-light`, offers Telegram or Discord, asks for the chosen bot token, administrator user ID, and target game, then builds and starts the container. Docker must already be installed; the script stops with an install hint if Docker or Compose is missing. After startup, upload any `.txt` document to Telegram or use Discord `/accounts file`;  it is saved as `account.txt` after validation (maximum 2 MiB).

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
2. Run `start.bat`. On first launch, it creates local configuration files and offers Telegram or Discord, then prompts for the chosen bot token, administrator user ID, and game. It then creates a sample `account.txt` and exits so you can fill it in.
3. Add accounts to `account.txt` using the format below, then run `start.bat` again.
4. Keep the console open while the miner runs. The launcher builds and starts `tdm-light.exe`.

To replace the account list later, send any `.txt` document to Telegram or use Discord `/accounts file`. It is saved as `account.txt`.

### Bot selection and Discord setup

On the first interactive launch, choose **1 — Telegram** or **2 — Discord**. The choice is saved in `config.env`; complete existing Telegram settings are adopted automatically. Only the selected bot runs. The standalone executable includes the wizard (`tdm-light.exe --setup` on Windows or `./tdm-light --setup` on Linux); systemd/Docker startup never waits for console input. To switch later, change `BOT_PLATFORM=telegram` or `BOT_PLATFORM=discord`, run setup for missing credentials, and restart. For Docker, edit `.env` and recreate the container with `docker compose up -d`. `BOT_PLATFORM=none` explicitly disables bot control.

For Discord:

1. Create an application and bot in the [Discord Developer Portal](https://discord.com/developers/applications), then copy the bot token.
2. Install it on your server with `bot` and `applications.commands` scopes. Keep a shared server with the bot so you can open its profile and direct messages; server channel permissions are not used for control.
3. Enable Discord Developer Mode and copy your **User ID**. Enter it and the bot token in the wizard, or configure `DISCORD_BOT_TOKEN` and `DISCORD_ADMIN_ID`. No channel ID is needed; old `DISCORD_CHANNEL_ID` settings are ignored.
4. Keep the Interactions Endpoint URL empty: this bot receives commands over its outgoing Gateway connection. Message Content and other privileged intents are unnecessary. Commands are registered on connection; global commands may take time to appear.
5. Use `/status`, `/logs login`, `/game [name]`, `/restart`, or `/accounts file`. Open the bot’s direct messages and use these slash commands there. Choose a `.txt` attachment for `/accounts`. Commands from servers, group chats, and other users are rejected. Reward notifications are sent to your DM, which the bot resolves automatically. Allow direct messages from the shared server and unblock the bot if Discord blocks delivery. Simply sending a text message or an attachment without `/accounts` is not a command.

Bot network activity is independent of watch requests. Notification memory is bounded (128 entries); bursts are batched, excess notifications are counted in a summary, and delivery failures are logged. Complete reward details remain in account logs and `/status`. Discord reconnects with backoff and resumes its session when possible; API rate-limit waits are cancellable. This isolates bot outages from farming, but does not guarantee Twitch crediting or message delivery during outages.

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

Copy `tdm-light-vps-ubuntu.tar.gz` to the server, then run:

```bash
tar -xzf tdm-light-vps-ubuntu.tar.gz
cd vps-linux
chmod +x install.sh
sudo ./install.sh
```

The installer selects the `amd64` or `arm64` binary, offers Telegram or Discord and configures the target game, then creates and starts `tdm-light.service`. Build the bundle before copying it; binaries and archives are not stored in the source repository.

### 🤖 Telegram or Discord bot

The bot accepts commands from the configured administrator. Discord commands work only in the owner’s direct messages with the bot. Available commands include:

| Command | Action |
| --- | --- |
| `/status` | Show accounts, crediting speed, lagging accounts, recovery attempts, and ready rewards |
| `/logs <login>` | Show logs for an account |
| `/game` | Show or choose the target game |
| `/game <name>` | Change the target game |
| `/restart` | Restart the miner |

To update the account list, send any `.txt` document in Telegram or use `/accounts file` with an attachment in Discord. It is validated and saved as `account.txt`. The miner only accumulates watch time. At 100% it notifies you and moves on; rewards must be claimed manually.

`/logs <login>` shows confirmed minutes from Twitch Inventory, including unchanged progress. Missing personal progress is shown as unavailable, rather than zero. If HLS stays unavailable after a playlist refresh, the worker switches channels and excludes the failed channel for five minutes.

HLS playlist and segment headers are checked every 20 seconds. Minute-watched events use an independent interval (`HEARTBEAT_INTERVAL_SEC`, minimum 60 seconds); Inventory uses `CAMPAIGN_POLL_INTERVAL_SEC` (120 seconds by default). HLS checks do not download video. This increases HLS request frequency compared with the previous once-per-minute loop; improved crediting must be verified against live progress.

For 500 simultaneously active accounts, put them in the same group. Groups run in turn. HLS connections are reused; idle connections are limited to 128 total and 32 per host, with a 90-second idle timeout. There is no added global request queue. Each account keeps 100 log lines in memory and up to two 256 KiB log files. If TARGET_GAME is set in the process environment, change it there to select a different game.

Crediting speed uses confirmed Inventory progress over a 10–15 minute window with at least four observations. Persistently low progress (below 80% of elapsed time, allowing one minute for integer rounding) triggers a different channel or reconnection of the current channel. Recovery attempts are spaced 10–12 minutes apart. Missing data, long polling gaps, or decreasing counters reset the window.

`/status` averages fresh, full windows of actively farming accounts. It reports normal speed (at least 90%), lagging accounts, and accounts still collecting data, plus up to five slowest logins and the number of recovery attempts. After a channel/drop change or recovery, a fresh observation window is required. Catch-up updates from Twitch can temporarily produce rates above 100%. The status threshold of 90% is distinct from the more conservative automatic recovery rule.

Previous standard `accounts.txt` paths are normalized to `account.txt`. The old list is moved only when the new file is absent; an existing `account.txt` takes precedence. Explicit custom filenames remain supported. Duplicate logins, including case-only duplicates, are rejected.

### 🔐 Configuration and privacy

- `config.env.example` and `config.json.example` are configuration templates.
- Docker stores persistent data in `/opt/tdm-light/data` and bot secrets in `/opt/tdm-light/.env`.
- Never commit or share `account.txt`, `config.env`, `config.json`, `.env`, logs, or built binaries.
- Local account and configuration files are excluded from Git.

### 🧰 Development

```bash
go test ./...
go vet ./...
go build ./...
```

---

## Русский

TwitchDrops Miner Light — приложение на Go для просмотра Twitch-стримов с активными Drops по выбранной игре сразу с нескольких аккаунтов. Его можно запускать в Windows или развернуть на Ubuntu VPS через Docker Compose.

### ✨ Возможности

- 🎯 Ищет живые каналы с включёнными Drops для выбранной игры.
- 👥 Подключает несколько аккаунтов с распределённым стартом.
- ⏱ Читает подтверждённые минуты из Inventory; награды получает пользователь вручную.
- 📈 Показывает скорость начисления в Telegram или Discord и восстанавливает устойчиво медленный просмотр.
- 🤖 Выбор Telegram или Discord при первом запуске: статус, логи, выбор игры, перезапуск, уведомления о наградах и загрузка списка аккаунтов.
- 💾 Хранит аккаунты, логи и настройки игры в постоянном локальном каталоге.
- 🐳 Собирается и запускается на VPS через Docker Compose.

### 🚀 Быстрая установка на Ubuntu VPS

На Ubuntu VPS с установленными Docker Engine, плагином Docker Compose и доступом `sudo` выполни:

```bash
curl -fsSL https://raw.githubusercontent.com/Xyloni/tdm-light/main/deploy/bootstrap-docker.sh | sudo bash
```

Скрипт скачает проект в `/opt/tdm-light`, предложит Telegram или Discord, запросит токен выбранного бота, user ID администратора и целевую игру, затем соберёт и запустит контейнер. Docker должен быть установлен заранее; если Docker или Compose не найдены, скрипт завершится и попросит установить их. После запуска отправь любой `.txt` документом в Telegram или через `/accounts file` в Discord: после проверки он сохранится как `account.txt` (максимум 2 МиБ).

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
2. Запусти `start.bat`. При первом запуске он создаст локальные настройки, предложит Telegram или Discord, запросит токен выбранного бота, user ID администратора и игру, затем создаст образец `account.txt` и завершится, чтобы ты заполнил файл.
3. Добавь аккаунты в `account.txt` в формате ниже и запусти `start.bat` ещё раз.
4. Оставь консоль открытой, пока работает майнер. Батник соберёт и запустит `tdm-light.exe`.

Чтобы позже заменить список аккаунтов, отправь любой `.txt` документом в Telegram или через `/accounts file` в Discord: он сохранится как `account.txt`.

### Выбор бота и настройка Discord

При первом интерактивном запуске выбери **1 — Telegram** или **2 — Discord**. Выбор сохраняется в `config.env`; готовые настройки Telegram подхватываются автоматически. Запускается только выбранный бот. Мастер встроен в приложение: `tdm-light.exe --setup` в Windows или `./tdm-light --setup` в Linux. Запуск службы systemd/Docker не ожидает ввода. Для смены бота измени `BOT_PLATFORM=telegram` или `BOT_PLATFORM=discord`, запусти мастер для заполнения недостающих данных и перезапусти приложение. В Docker меняй `.env` и пересоздай контейнер: `docker compose up -d`. Значение `BOT_PLATFORM=none` отключает управление через бота.

Для Discord:

1. Создай приложение и бота в [Discord Developer Portal](https://discord.com/developers/applications), скопируй токен бота.
2. Добавь его на сервер со scope `bot` и `applications.commands`. Оставь общий сервер с ботом, чтобы открыть его профиль и личные сообщения. Права серверного канала для управления не используются.
3. Включи Developer Mode в Discord, скопируй свой **User ID**. Введи его и токен бота в мастере либо заполни `DISCORD_BOT_TOKEN` и `DISCORD_ADMIN_ID`. ID канала не требуется; прежний `DISCORD_CHANNEL_ID` игнорируется.
4. Оставь Interactions Endpoint URL пустым: команды поступают через исходящее Gateway-соединение. Message Content и другие привилегированные intents не нужны. Команды регистрируются при подключении; появление глобальных команд может занять время.
5. Используй `/status`, `/logs login`, `/game [name]`, `/restart`, `/accounts file`. Открой личку бота и используй slash-команды там. Для `/accounts` прикрепляй `.txt`. Команды с сервера, из групповых чатов и от других пользователей отклоняются. Уведомления о наградах поступают тебе в личку; бот определит её автоматически. Если Discord блокирует доставку, разреши личные сообщения с общего сервера и разблокируй бота. Обычный текст или вложение без `/accounts` не являются командой.

Запросы бота выполняются независимо от просмотра. Очередь уведомлений ограничена 128 записями; массовые уведомления объединяются, для переполнения выводится счётчик, ошибки доставки записываются в лог. Полные сведения о наградах остаются в логах аккаунтов и `/status`. Discord переподключается с увеличением пауз и по возможности восстанавливает сессию; ожидания ограничений API прерываются при остановке. Это изолирует фарм от сбоев бота, но не гарантирует начисления Twitch и доставку сообщений во время сбоев.

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

Скопируй `tdm-light-vps-ubuntu.tar.gz` на сервер и выполни:

```bash
tar -xzf tdm-light-vps-ubuntu.tar.gz
cd vps-linux
chmod +x install.sh
sudo ./install.sh
```

Установщик выберет бинарник `amd64` или `arm64`, предложит Telegram или Discord и настроит целевую игру, затем создаст и запустит службу `tdm-light.service`. Перед копированием собери комплект: готовые бинарники и архивы не хранятся в Git-репозитории.

### 🤖 Telegram- или Discord-бот

Бот принимает команды от заданного администратора. В Discord управление доступно только в личке бота. Доступны:

| Команда | Действие |
| --- | --- |
| `/status` | Показать аккаунты, скорость начисления, отстающих, попытки восстановления и готовые награды |
| `/logs <логин>` | Показать логи аккаунта |
| `/game` | Показать или выбрать целевую игру |
| `/game <название>` | Сменить целевую игру |
| `/restart` | Перезапустить майнер |

Чтобы обновить список аккаунтов, отправь любой `.txt` документом в Telegram или прикрепи его к `/accounts file` в личке Discord-бота. Он сохранится как `account.txt`. Программа только набирает время просмотра. При 100% она уведомит вас и перейдёт дальше; получать награды нужно вручную.

`/logs <логин>` показывает подтверждённые минуты из Twitch Inventory, в том числе когда значение не изменилось. Отсутствие личных данных отображается отдельно, без подмены нулём. Если HLS не восстановился после обновления плейлиста, воркер меняет канал и исключает проблемный на пять минут.

Плейлист HLS и заголовки сегментов проверяются каждые 20 секунд. Маяк minute-watched имеет независимый интервал (`HEARTBEAT_INTERVAL_SEC`, минимум 60 секунд), Inventory — свой (`CAMPAIGN_POLL_INTERVAL_SEC`, по умолчанию 120 секунд). Видео не скачивается. Частота HLS-запросов выше прежнего минутного цикла; улучшение начисления нужно подтвердить живым прогоном.

Скорость начисления проверяется по подтверждённым минутам Inventory за окно от 10 минут (максимум 15), минимум по четырём наблюдениям. Если прирост ниже 80% длительности окна с допуском в одну минуту на округление, программа пробует другой канал. Если замены нет, переподключает текущий. Пауза между попытками — 10–12 минут на аккаунт, в том числе после неудачи. Пропавшие данные, длительные перерывы опроса и уменьшение счётчика сбрасывают окно. Причина восстановления записывается в логи; успешность улучшения проверяется живым прогоном.

`/status` показывает среднюю скорость по свежим полным окнам активно фармящих аккаунтов, количество аккаунтов с нормальной скоростью (от 90%), отстающих и ещё собирающих данные. Отображаются до пяти самых медленных логинов и число попыток восстановления скорости за запуск. Первое окно требует не менее 10 минут наблюдения; после смены канала, дропа или восстановления данные собираются заново. Просроченные данные исключаются из среднего. Скорость может временно превышать 100%, если Twitch сообщает догоняющее обновление. Порог «отстаёт» в сводке — 90%; автоматическое восстановление использует более консервативное правило, описанное выше.

Чтобы 500 аккаунтов работали одновременно, поместите их в одну группу: группы работают по очереди. Соединения HLS переиспользуются; сохраняется до 128 простаивающих соединений всего и до 32 на сервер, с закрытием через 90 секунд простоя. Общая очередь запросов не добавлена. На аккаунт хранится 100 строк логов в памяти и максимум два файла по 256 КиБ. Если TARGET_GAME задан в окружении процесса, менять игру нужно там.

Прежние стандартные пути `accounts.txt` нормализуются в `account.txt`. Старый список переносится только при отсутствии нового файла; существующий `account.txt` имеет приоритет. Нестандартные имена из конфига поддерживаются. Дубли логинов, в том числе отличающиеся только регистром, отклоняются.

### 🔐 Настройки и приватность

- `config.env.example` и `config.json.example` — шаблоны настроек.
- Docker хранит постоянные данные в `/opt/tdm-light/data`, а секреты бота — в `/opt/tdm-light/.env`.
- Не публикуй и не отправляй посторонним `account.txt`, `config.env`, `config.json`, `.env`, логи и собранные бинарники.
- Локальные файлы аккаунтов и настроек исключены из Git.

### 🧰 Разработка

```bash
go test ./...
go vet ./...
go build ./...
```
