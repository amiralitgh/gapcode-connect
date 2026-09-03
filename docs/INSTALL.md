# Install GapCode Connect

GapCode Connect links a local GapCode session to Telegram and Bale.

## Fresh install

Install and sign in to [GapCode](https://gapgpt.app/gapcode), then run:

```bash
git clone https://github.com/amiralitgh/gapcode-connect.git
cd gapcode-connect
./install.sh --setup
```

`install.sh` builds the connector. If Go is missing, it installs Go with
`apt` on Linux or Homebrew on macOS.

The wizard writes `~/.cc-connect/config.toml` and a private
`~/.cc-connect/secrets.env`. It checks the bot tokens and asks for separate
Telegram and Bale user IDs.

If you do not know an ID yet, keep the placeholder, start the connector, and
send `/whoami` from that platform. Put the returned ID into `config.toml` and
restart.

## Start

```bash
set -a
source ~/.cc-connect/secrets.env
set +a
./gapcode-connect --config ~/.cc-connect/config.toml
```

## Telegram proxy

The wizard uses `socks5://127.0.0.1:10808` by default. This is the usual local
V2Ray SOCKS5 port. Change or remove it if Telegram is reachable directly.

## Bale commands

Create the bot with [Bale BotFather](https://ble.ir/botfather), then paste
[`docs/bale-commands-en.txt`](docs/bale-commands-en.txt) into BotFather.
