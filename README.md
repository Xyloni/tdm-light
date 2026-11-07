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

On Ubuntu with Docker Engine, the Docker Compose plugin, and sudo access:

```bash
curl -fsSL https://raw.githubusercontent.com/Xyloni/tdm-light/main/deploy/install-vps.sh -o /tmp/tdm-light-install.sh
sudo bash /tmp/tdm-light-install.sh
```

The installer downloads a prebuilt amd64/arm64 image and creates `/opt/tdm-light-vps`. Choose Telegram or Discord and enter your bot token, user ID and target game. Upload a `.txt` account file through the bot (Discord uses `/accounts`). Configure automatic shutdown, diagnostics and logging through `/settings`. Persistent files stay in `data`.

Update manually after choosing a suitable time:

```bash
cd /opt/tdm-light-vps
sudo bash update.sh
```

| Action | Command |
| --- | --- |
| Follow logs | `cd /opt/tdm-light-vps && sudo docker compose -f compose.yaml logs -f tdm-light` |
| Start after automatic shutdown | `cd /opt/tdm-light-vps && sudo docker compose -f compose.yaml start` |
| Roll back | `cd /opt/tdm-light-vps && sudo bash update.sh --rollback` |

See [the VPS guide](vps/README.md) for version pinning, backups and requirements. Existing source-build installations can continue using `git pull` and `docker compose up -d --build` in their own directory.

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
| `/settings` | Edit settings with buttons, apply now or after the current group |
| `/status` | Show accounts, crediting speed, lagging accounts, recovery attempts, and ready rewards |
| `/logs <login>` | Show logs for an account |
| `/game` | Show or choose the target game |
| `/game <name>` | Change the target game |
| `/exclude [name]` | List exclusions, or exclude a reward by full name |
| `/include <name>` | Return a reward to farming |
| `/restart` | Restart the miner |

To update the account list, send any `.txt` document in Telegram or use `/accounts file` with an attachment in Discord. It is validated and saved as `account.txt`. The miner only accumulates watch time. At 100% it notifies you and moves on; rewards must be claimed manually.

`/logs <login>` shows confirmed minutes from Twitch Inventory, including unchanged progress. Missing personal progress is shown as unavailable, rather than zero. If HLS stays unavailable after a playlist refresh, the worker switches channels. Transient network errors, timeouts, HTTP 408/429/5xx receive a staggered 20–30 second cooldown; HTTP 404/403 and persistently slow channels retain the five-minute cooldown.

HLS playlist and segment headers are checked every 20 seconds. Minute-watched events use an independent interval (`HEARTBEAT_INTERVAL_SEC`, minimum 60 seconds); Inventory uses `CAMPAIGN_POLL_INTERVAL_SEC` (120 seconds by default). HLS checks do not download video. This increases HLS request frequency compared with the previous once-per-minute loop; improved crediting must be verified against live progress.

HLS, minute-watched beacons, and Inventory now execute independently. A slow request in one path does not hold up requests in the other two. Each active account has two additional sequential loops; there is at most one operation of each type in flight and only one pending Inventory result. Signal loops are cancelled and joined before changing drops or reloading.

A complete HLS check (including playlist refresh/retry) has a 15-second budget. Opening/recovering a channel has a 20-second total budget, with up to 8 seconds per candidate; beacon and Inventory operations each have 15 seconds. Failed HLS triggers another suitable streamer, with cooldown duration chosen by the error type. When all candidates are excluded, the next attempt follows the earliest cooldown expiry with jitter; directory retry pauses during long exclusions are capped at 60–63 seconds. Repeated directory failures use bounded 15–70 second backoff. When no channel can be opened, Inventory continues during the retry pause; beacons wait for a selected stream. Stream replacement cancels the old beacon request without resetting its cadence. These budgets bound attempts, not the gap between successful checks during a Twitch outage. HLS logs include the operation duration.

For 500 simultaneously active accounts, put them in the same group. Groups run in turn, without a session timer: the next group starts after all workers in the current group finish their available rewards. Legacy `GROUP_SESSION_MINUTES` settings are ignored, including existing values of 60. Each account advances between rewards independently. An authorization failure keeps the group waiting for corrected accounts and `/restart`; it is not treated as completed farming. HLS connections are reused; idle connections are limited to 128 total and 32 per host, with a 90-second idle timeout. There is no added global request queue. Each account keeps 100 log lines in memory. The default file log is shared at `logs/tdm-light.log`. If TARGET_GAME is set in the process environment, change it there to select a different game.

Crediting speed uses confirmed Inventory progress over a 10–15 minute window with at least four observations. Persistently low progress (below 80% of elapsed time, allowing one minute for integer rounding) triggers a different channel or reconnection of the current channel. Recovery attempts are spaced 10–12 minutes apart. Missing data, long polling gaps, or decreasing counters reset the window.

`/status` averages fresh, full windows of actively farming accounts. It reports normal speed (at least 90%), lagging accounts, and accounts still collecting data, plus up to five slowest logins and the number of recovery attempts. A drop change requires a fresh observation window; channel reconnects remain part of the overall status rate. Catch-up updates from Twitch can temporarily produce rates above 100%. The status threshold of 90% is distinct from the more conservative automatic recovery rule.

Previous standard `accounts.txt` paths are normalized to `account.txt`. The old list is moved only when the new file is absent; an existing `account.txt` takes precedence. Explicit custom filenames remain supported. Duplicate logins, including case-only duplicates, are rejected.

Reward notifications are grouped by account group, campaign, and reward ID. Newly completed accounts are summarized at most once per minute per reward/group; a reward completed by the whole group triggers a final summary promptly. Account details stay in `/logs`. Rewards already complete or claimed when first discovered are counted silently, including after reload/restart. Only a transition from confirmed unfinished minutes triggers a new completion notice. Identical reward names are distinguished by their target minutes. `/status` counts discovered ready rewards rather than implying they were all earned during the current run.

### Exclude unwanted rewards

Use `/exclude Hunt White` to exclude a reward, `/include Hunt White` to restore it, and `/exclude` to view the list. In Discord, choose the slash-command `name` option. Names match in full, case-insensitively, across all campaigns. Saving an exclusion reloads group processing. Excluded rewards are not farmed, counted as completed, or allowed to hold up a completed group.

`Hunt White` is excluded by default. Configure `IGNORED_DROP_NAMES=Hunt White, Other reward` in `config.env` (Docker: `data/config.env`), or `"ignored_drop_names": ["Hunt White"]` in `config.json`. An empty environment-file value or empty JSON array enables all rewards. Commands save the list as a JSON array in `config.env` to preserve names containing commas. An OS/container `IGNORED_DROP_NAMES` variable overrides the file and prevents bot edits until removed. An unknown progress value for any non-excluded reward remains unknown, rather than being treated as completed.

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

На Ubuntu VPS с установленными Docker Engine, плагином Compose и доступом sudo:

```bash
curl -fsSL https://raw.githubusercontent.com/Xyloni/tdm-light/main/deploy/install-vps.sh -o /tmp/tdm-light-install.sh
sudo bash /tmp/tdm-light-install.sh
```

Установщик скачает готовый образ amd64/arm64 и создаст `/opt/tdm-light-vps`. Выбери Telegram или Discord, введи токен, свой user ID и игру. Загрузи `.txt` через бота (Discord использует `/accounts`). Автовыключение, диагностика и логирование меняются через `/settings`. Постоянные файлы хранятся в `data`.

Обновление запускается вручную в выбранное тобой время:

```bash
cd /opt/tdm-light-vps
sudo bash update.sh
```

| Действие | Команда |
| --- | --- |
| Логи | `cd /opt/tdm-light-vps && sudo docker compose -f compose.yaml logs -f tdm-light` |
| Запуск после автовыключения | `cd /opt/tdm-light-vps && sudo docker compose -f compose.yaml start` |
| Откат | `cd /opt/tdm-light-vps && sudo bash update.sh --rollback` |

Подробности, закрепление версии и резервное копирование описаны в [инструкции VPS](vps/README.md). Прежние установки из исходников можно обновлять через `git pull` и `docker compose up -d --build` в их собственном каталоге.

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
| `/exclude [название]` | Показать исключения или исключить награду по полному названию |
| `/include <название>` | Вернуть награду в фарм |
| `/restart` | Перезапустить майнер |

Чтобы обновить список аккаунтов, отправь любой `.txt` документом в Telegram или прикрепи его к `/accounts file` в личке Discord-бота. Он сохранится как `account.txt`. Программа только набирает время просмотра. При 100% она уведомит вас и перейдёт дальше; получать награды нужно вручную.

`/logs <логин>` показывает подтверждённые минуты из Twitch Inventory, в том числе когда значение не изменилось. Отсутствие личных данных отображается отдельно, без подмены нулём. Если HLS не восстановился после обновления плейлиста, воркер меняет канал. Для сетевых ошибок, таймаутов и HTTP 408/429/5xx действует короткая пауза 20–30 секунд с разбросом. HTTP 404/403 и устойчиво медленные каналы сохраняют пятиминутное исключение.

Плейлист HLS и заголовки сегментов проверяются каждые 20 секунд. Маяк minute-watched имеет независимый интервал (`HEARTBEAT_INTERVAL_SEC`, минимум 60 секунд), Inventory — свой (`CAMPAIGN_POLL_INTERVAL_SEC`, по умолчанию 120 секунд). Видео не скачивается. Частота HLS-запросов выше прежнего минутного цикла; улучшение начисления нужно подтвердить живым прогоном.

HLS, маяк minute-watched и Inventory теперь выполняются независимо: медленный запрос одного типа не задерживает запросы двух других. На активный аккаунт добавлены два последовательных цикла; одновременно выполняется максимум одна операция каждого типа, хранится только последний ожидающий результат Inventory. Перед переходом на другую награду и перезагрузкой циклы отменяются, программа дожидается их завершения.

Вся HLS-проверка, включая обновление плейлиста и повтор, ограничена 15 секундами. На открытие/восстановление канала отводится суммарно 20 секунд, на отдельного кандидата — до 8 секунд; на маяк и Inventory — по 15 секунд. При сбое HLS программа пробует другого подходящего стримера, выбирая паузу по типу ошибки. Если все кандидаты временно исключены, повтор учитывает ближайшее окончание паузы и случайный разброс; при длинных исключениях пауза перед проверкой каталога ограничена 60–63 секундами. Повторные ошибки каталога получают ограниченный backoff 15–70 секунд. Если канал открыть не удалось, Inventory продолжает работать во время паузы перед повтором; маяки ждут выбранного стрима. Смена стримера отменяет незавершённый маяк старого канала, сохраняя расписание. Это бюджеты попыток, а не гарантия интервала между успешными проверками при сбое Twitch. В HLS-логах записывается длительность операции.

Скорость начисления проверяется по подтверждённым минутам Inventory за окно от 10 минут (максимум 15), минимум по четырём наблюдениям. Если прирост ниже 80% длительности окна с допуском в одну минуту на округление, программа пробует другой канал. Если замены нет, переподключает текущий. Пауза между попытками — 10–12 минут на аккаунт, в том числе после неудачи. Пропавшие данные, длительные перерывы опроса и уменьшение счётчика сбрасывают окно. Причина восстановления записывается в логи; успешность улучшения проверяется живым прогоном.

`/status` показывает среднюю скорость по свежим полным окнам активно фармящих аккаунтов, количество аккаунтов с нормальной скоростью (от 90%), отстающих и ещё собирающих данные. Отображаются до пяти самых медленных логинов и число попыток восстановления скорости за запуск. Первое окно требует не менее 10 минут наблюдения; после смены дропа данные собираются заново; переподключения входят в общую оценку скорости. Просроченные данные исключаются из среднего. Скорость может временно превышать 100%, если Twitch сообщает догоняющее обновление. Порог «отстаёт» в сводке — 90%; автоматическое восстановление использует более консервативное правило, описанное выше.

Чтобы 500 аккаунтов работали одновременно, поместите их в одну группу: группы работают по очереди без таймера: следующая запускается после завершения всех доступных наград текущей группой. Для кампании из четырёх наград каждый аккаунт должен набрать минуты всех четырёх; забирать их вручную перед переходом не требуется. Прежний `GROUP_SESSION_MINUTES`, включая значение 60, игнорируется. При ошибке авторизации группа ждёт исправления аккаунтов и `/restart`; ошибка не считается завершением фарма. Соединения HLS переиспользуются; сохраняется до 128 простаивающих соединений всего и до 32 на сервер, с закрытием через 90 секунд простоя. Общая очередь запросов не добавлена. На аккаунт хранится 100 строк логов в памяти. По умолчанию файловый лог общий: `logs/tdm-light.log`. Если TARGET_GAME задан в окружении процесса, менять игру нужно там.

Прежние стандартные пути `accounts.txt` нормализуются в `account.txt`. Старый список переносится только при отсутствии нового файла; существующий `account.txt` имеет приоритет. Нестандартные имена из конфига поддерживаются. Дубли логинов, в том числе отличающиеся только регистром, отклоняются.

Уведомления о наградах объединяются по группе аккаунтов, кампании и ID награды. Новые завершения суммируются не чаще раза в минуту на награду/группу; когда награда готова у всей группы, итоговая сводка сразу ставится в очередь отправки. Данные отдельных аккаунтов остаются в `/logs`. Награды, которые при первом обнаружении уже готовы или получены, учитываются молча, в том числе после перезагрузки и перезапуска. Новое уведомление создаётся только при переходе от подтверждённого незавершённого прогресса к готовому. Одинаковые названия различаются целевыми минутами. Счётчик `/status` показывает обнаруженные готовые награды и не означает, что все они заработаны за текущий запуск.

### Исключение ненужных наград

`/exclude Hunt White` исключает награду, `/include Hunt White` возвращает её в фарм, `/exclude` показывает список. В Discord укажи название в параметре slash-команды `name`. Сопоставление — по полному названию без учёта регистра, для всех кампаний. Сохранение исключения перезапускает обработку групп. Исключённые награды не фармятся, не считаются завершёнными и не удерживают уже завершённую группу.

По умолчанию исключён `Hunt White`. Можно задать `IGNORED_DROP_NAMES=Hunt White, Other reward` в `config.env` (Docker: `data/config.env`) либо `"ignored_drop_names": ["Hunt White"]` в `config.json`. Пустое значение в env-файле или пустой JSON-массив разрешают все награды. Команды сохраняют список как JSON-массив в `config.env`, поэтому названия с запятыми не разбиваются. Переменная `IGNORED_DROP_NAMES` в окружении ОС/контейнера перекрывает файл и запрещает правки через бота, пока её не удалят. Неизвестный прогресс любой другой награды по-прежнему не считается завершением.

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


### Test diagnostics / Диагностика тестов

**English:** Set `DIAGNOSTICS=true` and `SINGLE_LOG=true` in `config.env`, then restart. All account and service messages are appended to **`logs/tdm-light.log`** (or `LOG_DIR/tdm-light.log`), including account names. HTTP diagnostics record operation, host, status, protocol, request/connection/DNS/TCP/TLS timings, connection reuse and active requests at request start. Successful segment HEAD requests are summarized every minute and at shutdown. HTTP errors remain individual records. The `diagnostic hls_retry queue_ms` record measures admission delay separately from the request timeout. Runtime memory, goroutine and GC metrics are recorded every 30 seconds. No request URLs, headers or bodies are included in HTTP diagnostics. CPU usage is not measured. Restore `DIAGNOSTICS=false` after testing. Logging settings take effect on restart/reload.

The combined file **does not rotate or truncate** and grows throughout testing and subsequent runs. Stop the process before moving/deleting it or archiving the full test. A bounded asynchronous queue keeps disk/console waits out of farming; if it fills, dropped records are explicitly reported as `log_queue_dropped`. Abrupt termination can lose the buffered tail. `SINGLE_LOG=false` restores the previous per-account rotating files; `/logs` continues to show the in-memory account history in either mode.

Temporary HLS failures retry the current channel twice with staggered delays before switching on the third failure; beacons and progress polling continue. Permanent failures can switch immediately. Channel discovery/opening has a separate limit of 32 operations. A burst of 16 temporary HLS failures within 10 seconds limits new HLS retries across the instance to 32 in flight, then 64 after 30 seconds without a fresh burst. The retry limit expires after 60 seconds without a fresh burst. First checks and healthy streams bypass this limit. Successfully checked segments are remembered within the latest playlist window and are not checked again on overlapping playlists. The overall status rate includes outages across channel switches; the channel recovery detector keeps a separate observation window.

**Русский:** Для теста укажите `DIAGNOSTICS=true` и `SINGLE_LOG=true` в `config.env`, затем перезапустите программу. Все сообщения аккаунтов и сервиса дописываются в **`logs/tdm-light.log`** (или `LOG_DIR/tdm-light.log`) с отметками аккаунтов. Диагностика HTTP записывает операцию, хост, статус, протокол, время запроса/подключения/DNS/TCP/TLS, переиспользование соединения и число активных запросов на момент начала. Успешные запросы сегментов объединяются в сводку раз в минуту и при остановке. Ошибки HTTP записываются отдельно. Запись `diagnostic hls_retry queue_ms` показывает ожидание повтора отдельно от сетевого таймаута. Каждые 30 секунд записываются память, число горутин и показатели GC. URL, заголовки и тела запросов в HTTP-диагностику не попадают. Использование CPU не измеряется. После теста верните `DIAGNOSTICS=false`. Настройки логирования применяются при перезапуске/перезагрузке.

Общий файл **не обрезается и не ротируется**, а растёт во время теста и следующих запусков. Перед переносом, удалением или архивацией полного теста остановите программу. Ограниченная асинхронная очередь исключает ожидание диска/консоли из фарма; при переполнении число потерянных записей явно отмечается как `log_queue_dropped`. Аварийное завершение может потерять хвост буфера. `SINGLE_LOG=false` возвращает прежние отдельные файлы аккаунтов с ротацией; `/logs` в обоих режимах показывает историю аккаунта из памяти.

При временном сбое HLS программа дважды повторяет проверку текущего канала с разбросом задержек и переключается после третьего сбоя; маяки и опрос прогресса продолжаются. При постоянной ошибке переключение возможно сразу. Поиск и открытие каналов имеют отдельный лимит в 32 операции. После 16 временных ошибок HLS за 10 секунд программа допускает до 32 одновременных повторов HLS, а спустя 30 секунд без нового всплеска повышает лимит до 64. Через 60 секунд без нового всплеска лимит снимается. Первые проверки и исправно работающие потоки проходят без этого ограничения. Проверенные сегменты запоминаются в пределах последнего окна плейлиста и повторно не проверяются при перекрытии плейлистов. Общая скорость в статусе учитывает простои между каналами; для восстановления медленного канала используется отдельное окно.


Persistent moderate loss / Устойчивое умеренное отставание: in addition to the existing 10–15 minute detector, a 30–35 minute window checks accrual below 90% with a one-minute rounding allowance. Recovery requires the recent 10-minute window to still be slow; an earlier outage followed by normal accrual does not trigger this rule. Both rules share the 10–12 minute recovery cooldown. Channel/drop changes and invalid observations reset the recovery windows; overall status history still spans channel reconnects. The recovery log names the short or long window. Each long history retains about 16 observations at the default two-minute polling interval.

В дополнение к прежнему контролю за 10–15 минут добавлено окно 30–35 минут: начисление ниже 90% с допуском в одну минуту на округление. Для восстановления требуется, чтобы последнее короткое окно тоже оставалось медленным; прежний сбой с последующим нормальным начислением сам по себе не вызывает переключение. Оба правила используют общую паузу между восстановлениями 10–12 минут. Смена канала/награды и некорректные данные сбрасывают окна восстановления; общая скорость в статусе по-прежнему учитывает переподключения. В логе отмечается короткое или длительное окно. При стандартном опросе раз в две минуты длинная история хранит около 16 наблюдений.


### Automatic shutdown / Автовыключение

**English:** Set `STOP_WHEN_IDLE=true` and `IDLE_CHECK_ROUNDS=3` in `config.env` (Docker uses `data/config.env`). After farming finishes, the miner performs three consecutive full checks of every account group with no available unfinished reward, then exits normally. The count is configurable from 1 to 100. Finding a reward or encountering an error resets confirmations. Checks use `CAMPAIGN_POLL_INTERVAL_SEC` with a minimum cycle interval of 60 seconds and retain the usual staggered account startup. A cycle that performed farming does not count as an empty confirmation. With the option disabled (default), the miner continues checking for new rewards. An initially empty account list keeps waiting for upload.

The selected bot receives a shutdown message with the reason, check count, and account/group counts. Shutdown waits up to 30 seconds for sending, including any in-flight notification. A delivery failure is logged and shutdown proceeds. Bot control stops together with the miner. Start the program/service again to resume. Docker Compose now uses `restart: on-failure`, matching the VPS service, so a clean automatic exit stays stopped. Existing Docker deployments must recreate the container to apply the restart policy.

**Русский:** Укажите `STOP_WHEN_IDLE=true` и `IDLE_CHECK_ROUNDS=3` в `config.env` (для Docker — в `data/config.env`). После завершения фарма майнер выполняет три последовательных полных проверки всех групп без доступных незавершённых наград и штатно завершает работу. Количество проверок можно задать от 1 до 100. Обнаружение награды или ошибка сбрасывают счётчик. Используется `CAMPAIGN_POLL_INTERVAL_SEC` с минимальным интервалом цикла 60 секунд и обычным распределением запуска аккаунтов. Цикл, в котором выполнялся фарм, не считается пустой проверкой. По умолчанию опция отключена, программа продолжает искать новые награды. При изначально пустом списке аккаунтов программа ожидает загрузку.

Выбранный бот получает сообщение с причиной автовыключения, числом проверок, аккаунтов и групп. Ожидание отправки ограничено 30 секундами, включая уже выполняющуюся отправку. При недоступности бота ошибка записывается в лог и остановка продолжается. Бот останавливается вместе с майнером. Для продолжения запустите программу или службу заново. Docker Compose использует `restart: on-failure`, как служба VPS, чтобы штатное завершение оставляло контейнер остановленным. Для существующего контейнера нужно применить политику перезапуска через пересоздание.


### Bot settings / Настройки бота

Use `/settings` in Telegram or the Discord bot DM. Buttons edit `STOP_WHEN_IDLE`, `IDLE_CHECK_ROUNDS` (1–100), `DIAGNOSTICS` and `SINGLE_LOG`. Values are saved atomically in `config.env`; active and saved values are shown separately when they differ. Choose **Apply now** to reload groups immediately, or **After group** to wait for the current group to finish. Bot credentials are never displayed. Process environment overrides are reported as locked. Game and reward exclusions use the existing `/game`, `/exclude` and `/include` commands. Text input is supported as `/settings KEY VALUE` (Discord: `/settings action:KEY VALUE`).

Команда `/settings` доступна в Telegram и личке Discord. Кнопки меняют `STOP_WHEN_IDLE`, `IDLE_CHECK_ROUNDS` (1–100), `DIAGNOSTICS` и `SINGLE_LOG`. Значения атомарно сохраняются в `config.env`; при отличии отображаются активное и сохранённое значения. Кнопка **Применить сейчас** перезагружает обработку групп, **После группы** ждёт завершения текущей группы. Токены и ID владельца в меню не показываются. Параметры, закреплённые окружением процесса, помечаются как недоступные для изменения. Игра и исключения используют `/game`, `/exclude` и `/include`. Для текстового ввода используется `/settings KEY VALUE` (Discord: `/settings action:KEY VALUE`). Сохранённые значения также применяются при следующем запуске программы.

### Prebuilt VPS installer / VPS-установщик готового образа

The image-based installer, manual updates and rollback are documented in [vps/README.md](vps/README.md). Credentials and bot-editable settings stay in `data/config.env`. GitHub publishes amd64/arm64 images only after successful checks. The deployment uses no automatic updater.

Установка готового образа, ручное обновление и откат описаны в [vps/README.md](vps/README.md). Данные бота и редактируемые настройки хранятся в `data/config.env`. Образы amd64/arm64 публикуются GitHub после успешных проверок. Обновления запускаются вручную.
