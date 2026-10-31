# tdm light for Ubuntu Server

This standalone bundle targets Ubuntu Server 26.04.1 on `amd64` and `arm64`. It includes prebuilt Go binaries, so Go does not need to be installed on the VPS.

## Install

1. Copy the whole `vps-linux` folder to the server, for example with `scp` or SFTP.
2. Connect over SSH and enter that folder.
3. Run:

   ```bash
   chmod +x install.sh
   sudo ./install.sh
   ```

The installer prompts for the Telegram bot token (input is hidden), administrator Telegram ID, and target game. Press Enter to keep the existing game. It creates the restricted `tdm-light` system user and starts `tdm-light.service`.

The service starts with an empty account list and waits for a bot upload. Send `accounts.txt` to the configured bot as a document. The bot validates and saves it under `/opt/tdm-light`, then reloads the miner.

## Operations

```bash
sudo systemctl status tdm-light
sudo journalctl -u tdm-light -f
sudo systemctl restart tdm-light
```

The bot supports `/status`, `/logs <login>`, `/game`, `/game <name>`, `/restart`, reward notifications, and uploading `accounts.txt`. `/game <name>` changes the target and reloads the miner. You can also edit `TARGET_GAME` in `/opt/tdm-light/config.env` and use `/restart`.

The account list and Telegram credentials are stored on the server with owner-only file permissions. Keep SSH access restricted to trusted administrators.

---

# tdm light для Ubuntu Server

Этот отдельный комплект предназначен для Ubuntu Server 26.04.1 на `amd64` и `arm64`. Внутри уже есть Linux-бинарники, поэтому устанавливать Go на VPS не нужно.

## Установка

1. Скопируй всю папку `vps-linux` на сервер, например через `scp` или SFTP.
2. Подключись по SSH и перейди в эту папку.
3. Выполни:

   ```bash
   chmod +x install.sh
   sudo ./install.sh
   ```

Установщик запросит токен Telegram-бота (ввод скрыт), Telegram ID администратора и целевую игру. Нажми Enter, чтобы оставить прежнюю игру. Установщик создаст системного пользователя `tdm-light` с ограниченными правами и запустит службу `tdm-light.service`.

Сначала служба запускается с пустым списком аккаунтов и ждёт загрузки через бота. Отправь настроенному боту `accounts.txt` документом. Бот проверит файл, сохранит его в `/opt/tdm-light` и перезапустит майнер.

## Управление

```bash
sudo systemctl status tdm-light
sudo journalctl -u tdm-light -f
sudo systemctl restart tdm-light
```

Бот поддерживает `/status`, `/logs <логин>`, `/game`, `/game <название>`, `/restart`, уведомления о наградах и загрузку `accounts.txt`. Команда `/game <название>` меняет игру и перезапускает майнер. Можно также изменить `TARGET_GAME` в `/opt/tdm-light/config.env` и выполнить `/restart`.

Список аккаунтов и настройки Telegram хранятся на сервере с доступом только владельца файла. Ограничь SSH-доступ доверенными администраторами.
