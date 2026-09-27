# Telegram Send

This is my first project using GoLang. This project provides a simple command-line tool to send messages, files, and images to a Telegram chat. It is built in Go and utilizes the Telegram Bot API for communication.

## Features

- Send text messages to a specified Telegram chat.
- Send files and images with optional captions.
- Send multiple files or images as Telegram albums.
- Read messages from stdin to send to a Telegram chat.
- Use a Bot API mirror or a proxy when api.telegram.org is not reachable.

## Usage

### Send a message

- With args:

  `telegram-send "hello world"`

- With stdin (pipe, no flags needed):

  `cat /var/log/log.txt | grep "hallo" | telegram-send`

- Force reading stdin (even if you're not piping):

  `telegram-send -stdin`

Messages longer than 4096 characters are split into several messages, at line breaks where possible.

### Send files and images

- Single document:

  `telegram-send -file ./report.pdf -caption "Monthly report"`

- Single image:

  `telegram-send -image ./photo.jpg -caption "Holiday"`

- Several documents as an album:

  `telegram-send -files ./a.pdf,./b.pdf -caption "Reports"`

- Several images as an album:

  `telegram-send -images ./1.jpg,./2.jpg,./3.jpg -caption "Trip"`

Notes:
- Only one of `-file`, `-files`, `-image`, `-images` can be used at a time.
- An album holds up to 10 items, so longer lists are sent as several albums. The caption is attached to the first item.
- Message text (args or stdin) is used as the caption when `-caption` is not set:

  `tail -n 20 app.log | telegram-send -file ./app.log`

- Captions longer than 1024 characters are sent as a separate message after the files.
- The Bot API accepts files up to 50 MB.

### Errors

Errors are printed to stderr and the tool exits with code 1. When a message or album list is sent in several parts, sending stops at the first failure and the error says which part failed.

## Configuration

The config is read from the first `config.yaml` found in `/etc/telegram-send/`, `./config` or the current directory. `config/config.yaml` in this repository is a template, keep your real token out of it.

```yaml
telegram:
  token: "123456:ABC..."   # Bot token from @BotFather
  chatid: "123456789"      # Chat to send to
  threadid: 123            # Optional: forum topic ID
  api_url: "https://tg.example.com"      # Optional: Bot API mirror, defaults to https://api.telegram.org
  proxy: "socks5://user:pass@host:1080"  # Optional: http://, https:// or socks5:// proxy
```

If `proxy` is not set, the `HTTPS_PROXY` / `HTTP_PROXY` environment variables are used.

### Bot API mirror

Where api.telegram.org is blocked, you can run a mirror on a server that can reach it and point `api_url` to it. A minimal nginx example:

```nginx
server {
    listen 443 ssl;
    server_name tg.example.com;
    # ssl_certificate ...

    access_log off;  # request URLs contain the bot token

    location / {
        proxy_pass https://api.telegram.org;
        proxy_set_header Host api.telegram.org;
        proxy_ssl_server_name on;
        client_max_body_size 50m;
    }
}
```

## Prerequisites

Before you can use this tool, you need to:

1. Create a Telegram bot by talking to [@BotFather](https://t.me/botfather) and get the bot token.
2. Find out the chat ID where you want to send messages.
