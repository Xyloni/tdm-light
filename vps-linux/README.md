# Dropfarm for Ubuntu Server

This bundle targets Ubuntu Server 26.04.1 on `amd64` and `arm64`. It includes prebuilt Go binaries; Go does not need to be installed on the VPS.

## Install

1. Copy the whole `vps-linux` folder to the server, for example with `scp` or SFTP.
2. Connect over SSH and enter that folder.
3. Run:

   ```bash
   chmod +x install.sh
   sudo ./install.sh
   ```

The installer prompts for the Telegram bot token (input is hidden), administrator Telegram ID, and target game. Press Enter to keep the existing game. It then creates a restricted `dropfarm` system user and starts the service.

The service starts with an empty account list and waits for the bot upload. Send `accounts.txt` to the configured bot as a document. The bot validates and saves it under `/opt/dropfarm`, then reloads the miner.

## Operations

```bash
sudo systemctl status dropfarm
sudo journalctl -u dropfarm -f
sudo systemctl restart dropfarm
```

The bot supports `/status`, `/logs <login>`, `/game`, `/game <name>`, `/restart`, reward notifications, and uploading `accounts.txt`. `/game <name>` changes the target and reloads the miner. You can also edit `TARGET_GAME` in `/opt/dropfarm/config.env` and use `/restart`.

The account list and Telegram credentials are stored on the server with owner-only file permissions. Keep SSH access restricted to trusted administrators.
