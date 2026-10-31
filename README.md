# Dropfarm

Dropfarm is a Go service for managing Twitch Drops watch sessions across multiple accounts. It can run on Windows or as a Docker container on an Ubuntu VPS. Account lists and Telegram credentials are local runtime data and are not part of the source repository.

## Features

- Runs account workers concurrently and staggers their initial start over the first minute.
- Selects live channels from the target game with Drops enabled.
- Sends minute watch heartbeats and refreshes an HLS playlist after a failed request.
- Claims completed rewards and notifies the configured Telegram administrator.
- Telegram controls: status, per-account logs, game selection, account-list upload, and reload.
- Docker Compose deployment with persistent account and log storage.

## Windows quick start

1. Install Go 1.22 or newer.
2. Run `start.bat`.
3. On first launch, enter the Telegram bot token, your numeric Telegram ID, and confirm or change the game.
4. If `accounts.txt` was missing, add accounts in the format below and run `start.bat` again.

The bot token is entered without echo. Existing settings are kept in the local `config.env` file.

## VPS quick start with Docker

Docker Engine and the Docker Compose plugin must be installed on the VPS. Once the repository is public, the bootstrap script can clone the project, prompt for first-run settings, build the image, and start the container in one command:

```bash
curl -fsSL https://raw.githubusercontent.com/Xyloni/tdm-light/main/deploy/bootstrap-docker.sh | sudo bash
```

The script clones this repository into `/opt/dropfarm`, then prompts for the Telegram bot token (hidden), Telegram administrator ID, and target game. The account list can be uploaded to the bot after it starts.

For a private repository, clone it using your normal Git credentials and run this single deployment command from the checkout:

```bash
bash ./deploy/install-docker.sh
```

The container uses the local `data/` directory for persistent account and log files. This also lets `/game <name>` save the selection across container restarts. The image supports `amd64` and `arm64` and builds from source; Go is not needed on the VPS.

Useful Docker commands:

```bash
docker compose logs -f dropfarm
docker compose ps
docker compose restart dropfarm
docker compose up -d --build
```

## Telegram bot

Only the configured administrator Telegram ID can use commands or upload the account list.

| Command / action | Result |
|---|---|
| `/status` | Account counts, active group, claims, and uptime |
| `/logs <login>` | Recent logs for one account |
| `/game` | Show the current target game |
| `/game <name>` | Save a new target game and reload the workers |
| `/restart` | Reload configuration and the account list |
| Send `accounts.txt` as a document | Validate, save, and reload the account list |

Completed rewards are also sent as Telegram notifications. Detailed campaign percentages are not exposed as a separate bot command; account logs show progress when Twitch returns an updated value.

## Account file format

One account per line, with six colon-separated fields:

```text
1:
email:login:password:oauth_token:user_ID:device_ID
email:login:password:oauth_token:user_ID:device_ID
```

Group markers such as `1:` are optional. Blank lines and lines beginning with `#` are ignored. The Telegram upload must be named `accounts.txt` and is limited to 2 MB.

## Configuration

`config.env.example` and `config.json.example` contain safe defaults. Windows uses local `config.env`. Docker keeps Telegram credentials in local `.env` and the selected game in `data/config.env`; all of these runtime files are excluded from Git.

| Setting | Purpose |
|---|---|
| `TARGET_GAME` | Game/category to farm |
| `TELEGRAM_BOT_TOKEN` | Bot token from BotFather |
| `TELEGRAM_ADMIN_ID` | Numeric Telegram user ID allowed to control the bot |
| `ACCOUNTS_FILE` | Account file path; Docker sets `/data/accounts.txt` |
| `HEARTBEAT_INTERVAL_SEC` | Watch heartbeat interval; default `60` |
| `CAMPAIGN_POLL_INTERVAL_SEC` | Campaign progress polling interval |
| `GROUP_SESSION_MINUTES` | Duration of each account group session; `0` disables rotation |

## Build and verify

```bash
go test ./...
go build .
```

The Windows launcher builds the program before starting it. `build-vps.ps1` creates Linux `amd64` and `arm64` binaries for the non-Docker VPS bundle.

## Runtime data

Never commit real `accounts.txt`, `config.env`, `config.json`, Docker `.env`, logs, or generated binaries. They are excluded by `.gitignore` and `.dockerignore`. Keep the VPS data volume and SSH access restricted because account files contain credentials.
