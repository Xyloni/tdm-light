<div align="center">

# TwitchDrops Miner Light

Lightweight multi-account Twitch Drops farming with Telegram or Discord control.

[English](#english) · [Русский](#русский)

Windows · Linux amd64 / arm64 · Docker Compose · Go

</div>

## English

TDM Light accumulates Twitch Drops watch time across multiple accounts for a selected game. It discovers suitable live channels, reads confirmed progress from Twitch Inventory, and controls account groups through Telegram or Discord. Rewards are claimed manually.

This README describes the current `main` branch and its container images. Downloadable release archives contain the code from their release tag.

### Features

- Concurrent accounts within a group, with startup spread over the first minute.
- Sequential groups that advance after every account finishes all available, non-excluded rewards.
- Independent HLS checks, minute-watched beacons and progress polling.
- Channel recovery for playback failures and sustained slow accrual, with shared protection against bursts of HLS retries.
- Owner-only bot commands, settings buttons, `.txt` account uploads and grouped reward notifications.
- Optional automatic shutdown with a final bot message.
- Persistent configuration and logs, prebuilt VPS images and manual updates with rollback support.

### Install on a VPS

Requirements are a Linux amd64/arm64 VPS, Docker Engine, the Docker Compose plugin, `curl`, `tar` and sudo access.

```bash
curl -fsSL https://raw.githubusercontent.com/Xyloni/tdm-light/main/deploy/install-vps.sh -o /tmp/tdm-light-install.sh
sudo bash /tmp/tdm-light-install.sh
```

The installer creates `/opt/tdm-light-vps`, downloads `ghcr.io/xyloni/tdm-light:latest`, and asks for the bot platform, token, owner user ID and target game. The token input is hidden. Existing account and configuration files are preserved when the installer is rerun.

After startup, upload an account file through the bot. Telegram accepts a `.txt` document. Discord uses `/accounts` with the `file` attachment option. Open `/settings` to configure automatic shutdown, diagnostics and logging.

The deployment directory contains these files.

| Path | Contents |
| --- | --- |
| `.env` | Image selection (`TDMLIGHT_IMAGE`) and timezone (`TZ`) |
| `data/config.env` | Bot credentials, game, exclusions and editable settings |
| `data/account.txt` | Account list |
| `data/logs/tdm-light.log` | Combined log when `SINGLE_LOG=true` |
| `.previous-image` | Previous image digest saved when an update changes the image |

Keep backups of `data` private. For a custom installation path, use `sudo TDMLIGHT_DIR=/your/absolute/path bash /tmp/tdm-light-install.sh`.

#### Manage, update and roll back

Run these commands from the installation directory.

```bash
cd /opt/tdm-light-vps
```

| Action | Command |
| --- | --- |
| Status | `sudo docker compose -f compose.yaml ps -a` |
| Follow container logs | `sudo docker compose -f compose.yaml logs -f tdm-light` |
| Stop | `sudo docker compose -f compose.yaml stop` |
| Start, including after automatic shutdown | `sudo docker compose -f compose.yaml start` |
| Restart | `sudo docker compose -f compose.yaml restart` |
| Update the selected image | `sudo bash update.sh` |
| Roll back the previous update | `sudo bash update.sh --rollback` |

Updates are manual. The script pulls the image before replacing the container and preserves `data`. It checks basic container startup and attempts to restore the previous image if startup fails and its digest is available. Twitch accrual and bot authentication require checking after startup.

`latest` follows successful builds from `main`. To select a specific build, replace `COMMIT_SHA` with a full published commit SHA.

```bash
sudo bash update.sh ghcr.io/xyloni/tdm-light:sha-COMMIT_SHA
```

To return to the latest successful build:

```bash
sudo bash update.sh ghcr.io/xyloni/tdm-light:latest
```

The container uses `restart: on-failure`. A normal automatic shutdown leaves it stopped. Bot control exits with the miner, so use the SSH `start` command to resume.

Additional deployment details are in [the VPS guide](vps/README.md). Existing source-build Docker installations can continue using `git pull` and `docker compose up -d --build` in their own directory. Rerun `deploy/install-docker.sh` to migrate editable options from the container `.env` into `data/config.env`.

### Run on Windows

For the current source version:

1. Install Go 1.22 or newer and download or clone this repository.
2. Run `start.bat`. Follow the first-run setup to choose Telegram or Discord and enter credentials.
3. If the launcher creates a sample `account.txt`, replace its rows with real accounts and run `start.bat` again.
4. Keep the console open while the miner runs. Use Ctrl+C for a graceful stop.

The launcher builds `tdm-light.exe`. A prebuilt executable from a release can be run directly without Go. Use `tdm-light.exe --setup` to fill missing setup values.

For a standalone Linux systemd deployment, build the bundle with `build-vps.ps1` and follow [the standalone Linux guide](vps-linux/README.md). The generated archive includes amd64 and arm64 binaries.

### Configure a bot

Choose one bot platform during setup. The miner runs the selected bot and accepts management actions from its configured owner.

**Telegram**

Create a bot through `@BotFather`, then enter its token and your numeric user ID during setup. Open the bot's private chat and send `/start`.

**Discord**

1. Create an application and bot in the [Discord Developer Portal](https://discord.com/developers/applications).
2. Copy the bot token from the **Bot** section. Enter it and your user ID during setup. Enable Developer Mode in Discord to copy your user ID.
3. Install the application on a shared server with the `bot` and `applications.commands` scopes. Allow direct messages from that server.
4. Leave **Interactions Endpoint URL** empty. Privileged intents are unnecessary for this implementation.
5. Open the bot's direct messages and use slash commands. Commands are registered on connection and may take time to appear.

Discord management works in the owner's DM. For account uploads, attach the document to `/accounts` using its `file` option.

To switch platforms later, change `BOT_PLATFORM` and fill the corresponding credentials in `config.env` (`data/config.env` on Docker), then restart the process. `BOT_PLATFORM=none` disables bot control. Credentials are configured outside `/settings`.

### Bot commands

| Command | Purpose |
| --- | --- |
| `/help` | Show available commands |
| `/settings` | Open editable settings and apply controls |
| `/status` | Show progress, accrual speed, lagging accounts and discovered ready rewards |
| `/logs <login>` | Show the recent in-memory history of an account |
| `/game` | Show the selected game |
| `/game <name>` | Save the game and reload group processing |
| `/exclude [name]` | List exclusions or exclude a reward by its full name |
| `/include <name>` | Restore an excluded reward |
| `/restart` | Reload configuration and the account list |
| `/accounts file` | Discord `.txt` attachment upload; Telegram accepts documents directly |

In Discord, use the corresponding slash-command options such as `login`, `name` and `file`.

#### Settings and when they apply

`/settings` provides buttons for these parameters.

| Parameter | Default | Meaning |
| --- | --- | --- |
| `STOP_WHEN_IDLE` | `false` | Exit after consecutive empty full checks |
| `IDLE_CHECK_ROUNDS` | `3` | Number of empty checks, from 1 to 100 |
| `DIAGNOSTICS` | `false` | Record HTTP diagnostics and runtime metrics |
| `SINGLE_LOG` | `true` | Write service and account messages to one combined file |

Changes are saved atomically. The menu shows saved and active values when they differ. Choose **Apply now** to reload group processing immediately, or **After group** to apply after the current group finishes. Saved values also apply on the next process start. Reloading rereads groups from the account file.

Text commands are also supported:

```text
/settings STOP_WHEN_IDLE true
/settings IDLE_CHECK_ROUNDS 3
/settings apply now
/settings apply after
```

For Discord, put the text after `/settings` in its `action` option, for example `/settings action:STOP_WHEN_IDLE true`.

An option set in the OS/container environment is locked against bot edits. In the prebuilt VPS deployment, keep editable options in `data/config.env`; the deployment `.env` holds image selection and timezone.

#### Automatic shutdown

With `STOP_WHEN_IDLE=true`, the miner finishes farming and then requires the configured number of consecutive full passes through all groups with no available unfinished reward. A pass that performed farming or encountered an error resets the counter. Authorization failures keep the affected group waiting for corrected accounts and `/restart`.

Checks follow `CAMPAIGN_POLL_INTERVAL_SEC` with a minimum cycle interval of 60 seconds, including the usual staggered account startup. An initially empty account list keeps waiting for upload.

When a bot is configured, the miner sends it a message before exiting with the reason and account/group counts. Sending has a total timeout of 30 seconds; a delivery failure is logged and shutdown proceeds. Start the program or container again to resume bot control and farming.

### Accounts and groups

Use one account per line, with six colon-separated fields. Supply the OAuth token without an `oauth:` prefix.

```text
email:login:password:oauth_token:user_ID:device_ID
```

Group markers go on separate lines:

```text
1:
email:login_one:password:oauth_token:user_ID:device_ID
email:login_two:password:oauth_token:user_ID:device_ID
2:
email:login_three:password:oauth_token:user_ID:device_ID
```

Without markers, all accounts belong to group 1. Groups run in numerical order. Accounts in a group farm concurrently and move between rewards independently. The next group waits until every account in the current group has finished all available, non-excluded rewards by confirmed minutes. Legacy `GROUP_SESSION_MINUTES` values are ignored.

Uploads accept any `.txt` filename, up to 2 MiB. The list is validated, saved as `account.txt`, and group processing reloads. Duplicate logins, including case-only duplicates, are rejected. Legacy `accounts.txt` is migrated when `account.txt` is absent.

### Reward exclusions and notifications

Use `/exclude Hunt White`, `/include Hunt White`, and `/exclude` to manage exclusions. Matching uses the full name, ignores case, and applies across campaigns. `Hunt White` is excluded by default. Saving an exclusion reloads group processing. Excluded rewards do not block a completed group.

File configuration supports a comma-separated list or JSON array:

```env
IGNORED_DROP_NAMES=["Hunt White", "Other reward"]
```

An empty value clears exclusions. Bot commands save a JSON array to preserve names containing commas. Missing Inventory progress for an eligible reward remains unknown until confirmed.

Completion notifications are grouped by account group, campaign and reward ID. Partial updates are summarized once per minute when new completions exist; completion by the whole group triggers a final summary promptly. Rewards already complete when discovered are counted silently. Claim rewards manually.

### Viewing, recovery and logs

HLS playlists and new segment headers are checked every 20 seconds. Beacons run independently at a minimum interval of 60 seconds. Inventory polling defaults to 120 seconds. Confirmed minutes can arrive with a delay, so a successful HTTP request alone does not establish successful accrual.

Temporary HLS failures retry the current channel twice with staggered delays before switching on the third failure. Permanent errors can trigger an earlier switch. Channel discovery/opening is limited to 32 concurrent recovery operations. During a burst of 16 temporary HLS failures within 10 seconds, new HLS retries are limited to 32 in flight, then 64 after 30 seconds without a fresh burst. The retry limit expires after 60 seconds without a fresh burst. First checks and healthy streams bypass this retry gate.

Slow accrual is checked over a 10–15 minute window and a separate 30–35 minute window. The longer rule targets sustained accrual below 90%, with a one-minute rounding allowance and a recent-window check. Recovery tries another channel or reconnects the current one. Attempts are spaced 10–12 minutes apart. `/status` retains overall accrual history across channel reconnects and needs a fresh observation window after changing rewards.

By default, all service and account records are appended to `logs/tdm-light.log`. On the prebuilt VPS deployment, this is `/opt/tdm-light-vps/data/logs/tdm-light.log`.

- `SINGLE_LOG=true` keeps one append-only file without rotation. Its size grows across runs.
- `SINGLE_LOG=false` restores per-account files with up to two 256 KiB generations per account.
- `/logs` shows up to 100 recent in-memory account records in either mode.
- `DIAGNOSTICS=true` adds HTTP phase timings, connection reuse, retry admission delays and runtime memory/goroutine/GC metrics. Successful segment requests are summarized every minute; errors remain individual records. CPU usage is not measured.
- The asynchronous log queue reports dropped records as `log_queue_dropped` if its capacity is exceeded. A graceful stop drains buffered records; abrupt termination can lose the tail.

Stop the miner before copying a complete test log. Disable diagnostics after testing. Signed request URLs are redacted; account identifiers and operational history remain in logs, so share them privately.

### Configuration and development

Configuration priority is **OS/container environment → `config.env` → `config.json` → defaults**. Templates are in [config.env.example](config.env.example) and [config.json.example](config.json.example).

| Parameter | Purpose |
| --- | --- |
| `BOT_PLATFORM` | `telegram`, `discord` or `none`; blank allows first-run selection |
| `TELEGRAM_BOT_TOKEN`, `TELEGRAM_ADMIN_ID` | Telegram credentials and owner ID |
| `DISCORD_BOT_TOKEN`, `DISCORD_ADMIN_ID` | Discord credentials and owner ID |
| `TARGET_GAME` | Selected game |
| `IGNORED_DROP_NAMES` | Excluded full reward names |
| `ACCOUNTS_FILE` | Account path; default `account.txt` |
| `HEARTBEAT_INTERVAL_SEC` | Beacon interval; default and minimum 60 seconds |
| `CAMPAIGN_POLL_INTERVAL_SEC` | Inventory interval; default 120, minimum 60 seconds |
| `LOG_DIR` | Log directory; default `logs` |
| `CLIENT_ID` | Twitch client ID matching the imported OAuth tokens; see the template |
| `JITTER_SEC` | Retry jitter setting; default 20 seconds |
| `USE_PUBSUB` | Optional event WebSocket; default `false` |

Keep account files, configuration containing credentials, logs and backups out of public repositories. The repository includes ignore rules for local runtime data.

```bash
go test ./...
go vet ./...
go build -o tdm-light .
```

GitHub checks Windows and Linux, runs the race detector on Linux, and checks installer data preservation and rollback with an isolated Docker command fixture. Container publication follows successful checks and a runtime smoke check. Images are published for Linux amd64 and arm64 under `latest` and `sha-COMMIT_SHA` tags. Live Twitch accrual depends on server, network and campaign conditions.

---

## Русский

TDM Light набирает время Twitch Drops на нескольких аккаунтах для выбранной игры. Программа находит подходящие прямые трансляции, получает подтверждённый прогресс из Twitch Inventory и управляет группами через Telegram или Discord. Награды забираются вручную.

README описывает текущую ветку `main` и её контейнерные образы. Готовые архивы релизов содержат код соответствующего релизного тега.

### Возможности

- Одновременная работа аккаунтов внутри группы с распределением запуска по первой минуте.
- Последовательные группы с переходом после завершения всех доступных наград с учётом исключений у каждого аккаунта.
- Независимые HLS-проверки, минутные маяки и опросы прогресса.
- Восстановление при сбоях просмотра и устойчивом отставании, общая защита от всплесков повторных HLS-проверок.
- Команды владельца, кнопки настроек, загрузка аккаунтов через `.txt` и группировка уведомлений.
- Настраиваемое автовыключение с итоговым сообщением боту.
- Постоянное хранение настроек и логов, готовые образы VPS и ручное обновление с поддержкой отката.

### Установка на VPS

Нужен Linux VPS с архитектурой amd64/arm64, Docker Engine, плагином Docker Compose, `curl`, `tar` и доступом sudo.

```bash
curl -fsSL https://raw.githubusercontent.com/Xyloni/tdm-light/main/deploy/install-vps.sh -o /tmp/tdm-light-install.sh
sudo bash /tmp/tdm-light-install.sh
```

Установщик создаёт `/opt/tdm-light-vps`, скачивает `ghcr.io/xyloni/tdm-light:latest` и запрашивает Telegram или Discord, токен, user ID владельца и игру. Ввод токена скрыт. Повторный запуск установщика сохраняет существующие аккаунты и конфигурацию.

После запуска загрузи файл аккаунтов через бота. В Telegram отправь `.txt` документом, в Discord прикрепи его к параметру `file` команды `/accounts`. Через `/settings` настрой автовыключение, диагностику и логирование.

В каталоге установки хранятся следующие файлы.

| Путь | Содержимое |
| --- | --- |
| `.env` | Выбор образа (`TDMLIGHT_IMAGE`) и часовой пояс (`TZ`) |
| `data/config.env` | Данные бота, игра, исключения и редактируемые настройки |
| `data/account.txt` | Список аккаунтов |
| `data/logs/tdm-light.log` | Общий лог при `SINGLE_LOG=true` |
| `.previous-image` | Digest прежнего образа, сохранённый при обновлении на другой образ |

Резервные копии `data` храни приватно. Другой каталог установки задаётся командой `sudo TDMLIGHT_DIR=/полный/путь bash /tmp/tdm-light-install.sh`.

#### Управление, обновление и откат

Выполняй команды из каталога установки.

```bash
cd /opt/tdm-light-vps
```

| Действие | Команда |
| --- | --- |
| Статус | `sudo docker compose -f compose.yaml ps -a` |
| Смотреть логи контейнера | `sudo docker compose -f compose.yaml logs -f tdm-light` |
| Остановить | `sudo docker compose -f compose.yaml stop` |
| Запустить, в том числе после автовыключения | `sudo docker compose -f compose.yaml start` |
| Перезапустить | `sudo docker compose -f compose.yaml restart` |
| Обновить выбранный образ | `sudo bash update.sh` |
| Откатить предыдущее обновление | `sudo bash update.sh --rollback` |

Обновление запускается вручную. Скрипт скачивает образ перед заменой контейнера и сохраняет `data`. Он проверяет запуск процесса и при неудаче пытается вернуть прежний образ, если доступен его digest. Начисление Twitch и авторизацию бота нужно проверять после запуска.

Тег `latest` соответствует успешным сборкам `main`. Для выбора конкретной сборки замени `COMMIT_SHA` полным SHA опубликованного коммита.

```bash
sudo bash update.sh ghcr.io/xyloni/tdm-light:sha-COMMIT_SHA
```

Вернуться на свежую успешную сборку:

```bash
sudo bash update.sh ghcr.io/xyloni/tdm-light:latest
```

Контейнер использует `restart: on-failure`. После штатного автовыключения он остаётся остановленным. Бот завершается вместе с майнером, поэтому для продолжения используй команду `start` по SSH.

Подробности — в [инструкции VPS](vps/README.md). Прежние Docker-установки из исходников можно обновлять через `git pull` и `docker compose up -d --build` в их собственном каталоге. Повторный запуск `deploy/install-docker.sh` переносит редактируемые параметры из контейнерного `.env` в `data/config.env`.

### Запуск на Windows

Для текущей версии из исходников:

1. Установи Go 1.22 или новее и скачай либо клонируй репозиторий.
2. Запусти `start.bat`. При первом запуске выбери Telegram или Discord и введи данные бота.
3. Если создан образец `account.txt`, замени его строки реальными аккаунтами и запусти `start.bat` повторно.
4. Оставь консоль открытой на время работы. Для штатной остановки используй Ctrl+C.

Загрузчик собирает `tdm-light.exe`. Готовый исполняемый файл из релиза запускается без Go. Команда `tdm-light.exe --setup` заполняет недостающие параметры первоначальной настройки.

Для отдельной Linux-службы systemd собери комплект через `build-vps.ps1` и следуй [инструкции standalone Linux](vps-linux/README.md). Созданный архив включает бинарники amd64 и arm64.

### Настройка бота

Выбери одну платформу при первоначальной настройке. Майнер запускает выбранного бота и принимает команды управления от заданного владельца.

**Telegram**

Создай бота через `@BotFather`, затем введи его токен и свой числовой user ID в мастере настройки. Открой личку бота и отправь `/start`.

**Discord**

1. Создай приложение и бота в [Discord Developer Portal](https://discord.com/developers/applications).
2. Скопируй токен из раздела **Bot**. Введи его и свой user ID при настройке. Чтобы скопировать user ID, включи Developer Mode в Discord.
3. Добавь приложение на общий сервер со scope `bot` и `applications.commands`. Разреши личные сообщения с этого сервера.
4. Оставь **Interactions Endpoint URL** пустым. Привилегированные intents для этой реализации не требуются.
5. Открой личку бота и используй slash-команды. Они регистрируются при подключении и могут появиться с задержкой.

Управление Discord доступно в личке владельца. Файл аккаунтов прикрепляется к параметру `file` команды `/accounts`.

Для смены платформы измени `BOT_PLATFORM` и заполни нужные данные в `config.env` (`data/config.env` в Docker), затем перезапусти процесс. Значение `BOT_PLATFORM=none` отключает управление ботом. Токены и ID настраиваются вне `/settings`.

### Команды бота

| Команда | Действие |
| --- | --- |
| `/help` | Список команд |
| `/settings` | Редактируемые настройки и кнопки применения |
| `/status` | Прогресс, скорость начисления, отстающие и обнаруженные готовые награды |
| `/logs <login>` | Последние записи аккаунта из памяти |
| `/game` | Показать выбранную игру |
| `/game <название>` | Сохранить игру и перезагрузить обработку групп |
| `/exclude [название]` | Список исключений или исключение награды по полному названию |
| `/include <название>` | Вернуть исключённую награду |
| `/restart` | Перечитать конфигурацию и список аккаунтов |
| `/accounts file` | Загрузка `.txt` через вложение Discord; Telegram принимает документы напрямую |

В Discord используй соответствующие параметры slash-команд, например `login`, `name` и `file`.

#### Настройки и момент применения

В `/settings` доступны кнопки следующих параметров.

| Параметр | По умолчанию | Значение |
| --- | --- | --- |
| `STOP_WHEN_IDLE` | `false` | Завершать работу после последовательных пустых полных проверок |
| `IDLE_CHECK_ROUNDS` | `3` | Количество пустых проверок от 1 до 100 |
| `DIAGNOSTICS` | `false` | Записывать диагностику HTTP и показатели процесса |
| `SINGLE_LOG` | `true` | Общий файл сообщений сервиса и аккаунтов |

Изменения сохраняются атомарно. Если сохранённое значение отличается от активного, меню показывает оба. Кнопка **Применить сейчас** сразу перезагружает обработку групп, **После группы** ждёт завершения текущей группы. Сохранённые значения применяются и при следующем запуске процесса. При перезагрузке группы заново читаются из файла аккаунтов.

Доступны и текстовые команды:

```text
/settings STOP_WHEN_IDLE true
/settings IDLE_CHECK_ROUNDS 3
/settings apply now
/settings apply after
```

В Discord текст после `/settings` указывается в параметре `action`, например `/settings action:STOP_WHEN_IDLE true`.

Параметр, заданный переменной окружения ОС или контейнера, заблокирован для изменения ботом. В VPS-установке готового образа редактируемые параметры хранятся в `data/config.env`, а установочный `.env` содержит выбор образа и часовой пояс.

#### Автовыключение

При `STOP_WHEN_IDLE=true` после фарма майнер ожидает заданное число последовательных полных обходов всех групп без доступных незавершённых наград. Цикл, в котором был фарм или ошибка, сбрасывает счётчик. При ошибке авторизации группа ожидает исправления аккаунтов и `/restart`.

Проверки следуют `CAMPAIGN_POLL_INTERVAL_SEC` с минимальным интервалом цикла 60 секунд и обычным распределением запуска аккаунтов. При изначально пустом списке программа ожидает загрузку.

Если бот настроен, перед выходом ему отправляется сообщение с причиной и количеством аккаунтов и групп. На отправку отводится до 30 секунд. При неудаче ошибка записывается в лог, после чего программа завершается. Для продолжения фарма и работы бота снова запусти программу или контейнер.

### Аккаунты и группы

Один аккаунт занимает одну строку из шести полей через двоеточие. OAuth-токен указывается без префикса `oauth:`.

```text
email:login:password:oauth_token:user_ID:device_ID
```

Маркеры групп располагаются на отдельных строках:

```text
1:
email:login_one:password:oauth_token:user_ID:device_ID
email:login_two:password:oauth_token:user_ID:device_ID
2:
email:login_three:password:oauth_token:user_ID:device_ID
```

Без маркеров все аккаунты входят в группу 1. Группы обрабатываются по возрастанию номера. Аккаунты внутри группы фармят одновременно и независимо переходят между наградами. Следующая группа ждёт подтверждённого набора минут всех доступных наград с учётом исключений у каждого аккаунта текущей группы. Старый параметр `GROUP_SESSION_MINUTES` игнорируется.

Загрузка поддерживает любое имя `.txt` размером до 2 МиБ. Список проверяется, сохраняется как `account.txt`, обработка групп перезагружается. Дубли логинов, в том числе отличающиеся регистром, отклоняются. Старый `accounts.txt` переносится, если `account.txt` отсутствует.

### Исключения наград и уведомления

Используй `/exclude Hunt White`, `/include Hunt White` и `/exclude` для управления исключениями. Сравнивается полное название без учёта регистра, правило действует во всех кампаниях. `Hunt White` исключён по умолчанию. Сохранение исключения перезагружает обработку групп. Исключённые награды не задерживают завершённую группу.

В конфигурации поддерживается список через запятую или JSON-массив:

```env
IGNORED_DROP_NAMES=["Hunt White", "Other reward"]
```

Пустое значение очищает исключения. Команды бота сохраняют JSON-массив, чтобы поддерживать названия с запятыми. Отсутствующий личный прогресс подходящей награды остаётся неизвестным до подтверждения.

Уведомления группируются по группе аккаунтов, кампании и ID награды. Промежуточная сводка отправляется раз в минуту при наличии новых завершений. Когда награду завершает вся группа, итоговая сводка ставится на отправку сразу. Уже завершённые при обнаружении награды учитываются молча. Получение наград остаётся ручным.

### Просмотр, восстановление и логи

HLS-плейлисты и заголовки новых сегментов проверяются каждые 20 секунд. Маяки работают независимо с минимальным интервалом 60 секунд. Опрос Inventory по умолчанию выполняется каждые 120 секунд. Подтверждённые минуты могут приходить с задержкой, поэтому успешный HTTP-запрос сам по себе не подтверждает начисление времени.

При временном сбое HLS программа дважды повторяет проверку текущего канала с разбросом задержек, после третьего сбоя переключает канал. Постоянные ошибки могут вызвать более раннее переключение. Поиск и открытие каналов ограничены 32 одновременными операциями восстановления. После 16 временных ошибок HLS за 10 секунд новые повторы HLS ограничиваются 32 одновременными операциями. Через 30 секунд без нового всплеска лимит повышается до 64, через 60 секунд снимается. Первые проверки и исправно работающие потоки проходят без этого ограничения повторов.

Скорость начисления проверяется за окно 10–15 минут и отдельное окно 30–35 минут. Длинное правило выявляет устойчивую скорость ниже 90% с допуском в одну минуту на округление и проверкой недавнего прогресса. Восстановление пробует другой канал или переподключает текущий. Между попытками выдерживается 10–12 минут. `/status` сохраняет общую историю начисления между переподключениями каналов. После смены награды требуется новое окно наблюдения.

По умолчанию сообщения сервиса и аккаунтов дописываются в `logs/tdm-light.log`. В VPS-установке готового образа путь — `/opt/tdm-light-vps/data/logs/tdm-light.log`.

- `SINGLE_LOG=true` сохраняет один дополняемый файл без ротации. Его размер растёт между запусками.
- `SINGLE_LOG=false` возвращает отдельные файлы аккаунтов с двумя поколениями до 256 КиБ каждое.
- `/logs` в обоих режимах показывает до 100 последних записей аккаунта из памяти.
- `DIAGNOSTICS=true` добавляет этапы HTTP-запросов, переиспользование соединений, ожидание повторов и показатели памяти, горутин и GC. Успешные запросы сегментов объединяются в сводку за минуту, ошибки записываются отдельно. Загрузка CPU не измеряется.
- При переполнении асинхронной очереди число потерянных строк отмечается как `log_queue_dropped`. Штатная остановка дописывает буфер. Аварийное завершение может потерять хвост.

Перед копированием полного тестового лога останови майнер. После теста отключи диагностику. Подписанные URL запросов скрываются, но логи содержат идентификаторы аккаунтов и историю работы, поэтому передавай их приватно.

### Конфигурация и разработка

Приоритет параметров — **окружение ОС/контейнера → `config.env` → `config.json` → значения по умолчанию**. Образцы находятся в [config.env.example](config.env.example) и [config.json.example](config.json.example).

| Параметр | Назначение |
| --- | --- |
| `BOT_PLATFORM` | `telegram`, `discord` или `none`; пустое значение позволяет выбрать при первом запуске |
| `TELEGRAM_BOT_TOKEN`, `TELEGRAM_ADMIN_ID` | Данные Telegram-бота и ID владельца |
| `DISCORD_BOT_TOKEN`, `DISCORD_ADMIN_ID` | Данные Discord-бота и ID владельца |
| `TARGET_GAME` | Выбранная игра |
| `IGNORED_DROP_NAMES` | Исключённые полные названия наград |
| `ACCOUNTS_FILE` | Путь списка аккаунтов, по умолчанию `account.txt` |
| `HEARTBEAT_INTERVAL_SEC` | Интервал маяка, по умолчанию и минимум 60 секунд |
| `CAMPAIGN_POLL_INTERVAL_SEC` | Интервал Inventory, по умолчанию 120, минимум 60 секунд |
| `LOG_DIR` | Каталог логов, по умолчанию `logs` |
| `CLIENT_ID` | Twitch Client ID, соответствующий импортированным OAuth-токенам; см. образец |
| `JITTER_SEC` | Настройка разброса повторных попыток, по умолчанию 20 секунд |
| `USE_PUBSUB` | Дополнительный WebSocket событий, по умолчанию `false` |

Не публикуй файлы аккаунтов, конфигурацию с токенами, логи и резервные копии. В репозитории предусмотрены правила исключения локальных данных.

```bash
go test ./...
go vet ./...
go build -o tdm-light .
```

GitHub проверяет Windows и Linux, запускает детектор гонок на Linux и проверяет сохранность данных установщика и откат с изолированным имитатором Docker-команд. Публикация контейнера следует за успешными проверками и проверкой запуска образа. Образы Linux amd64 и arm64 публикуются с тегами `latest` и `sha-COMMIT_SHA`. Реальное начисление Twitch зависит от сервера, сети и условий кампании.
