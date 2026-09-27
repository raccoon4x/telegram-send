# telegram-send

A small command-line tool that sends messages, files and images to a Telegram chat through the Bot API. Pipe anything into it from scripts, cron jobs or monitoring.

```sh
telegram-send "Deploy finished"
tail -n 50 app.log | telegram-send
telegram-send -images ./1.jpg,./2.jpg -caption "Screenshots"
```

## Features

- Text from arguments or stdin. Long text is split into several messages.
- Single files and images, or albums of several at once.
- Forum topics (`threadid`).
- Works where `api.telegram.org` is blocked, through your own Bot API mirror or a proxy.
- Script-friendly: errors go to stderr, and a non-zero exit code means the send failed.

## Quick start

1. Create a bot with [@BotFather](https://t.me/botfather) and copy its token.
2. Find the ID of the chat to send to. One way is to add the bot to the chat, send any message there and open `https://api.telegram.org/bot<TOKEN>/getUpdates`.
3. Build the binary (see [Building](#building)) and put it somewhere in your `PATH`.
4. Create `/etc/telegram-send/config.yaml`:

   ```yaml
   telegram:
     token: "123456:ABC..."
     chatid: "123456789"
   ```

5. Check that it works:

   ```sh
   telegram-send "hello"
   ```

## Usage

```
telegram-send [flags] [message text]
```

| Flag | Description |
| --- | --- |
| `-message <text>` | Message text. Positional arguments do the same. |
| `-stdin` | Read the message from stdin even when it is not a pipe. |
| `-file <path>` | Send a file as a document. |
| `-files <a,b,...>` | Send several files as document albums. |
| `-image <path>` | Send an image as a photo. |
| `-images <a,b,...>` | Send several images as photo albums. |
| `-caption <text>` | Caption for the files or images. |

### Text

```sh
telegram-send "hello world"
cat /var/log/syslog | grep error | telegram-send
```

When stdin is a pipe, it is read automatically. Text longer than 4096 characters is split into several messages, at line breaks where possible.

### Files and images

```sh
telegram-send -file ./report.pdf -caption "Monthly report"
telegram-send -image ./photo.jpg
telegram-send -files ./a.pdf,./b.pdf -caption "Reports"
telegram-send -images ./1.jpg,./2.jpg,./3.jpg -caption "Trip"
```

- Only one of `-file`, `-files`, `-image` and `-images` can be used at a time.
- If `-caption` is not set, the message text (arguments or stdin) becomes the caption:

  ```sh
  tail -n 20 app.log | telegram-send -file ./app.log
  ```

- Using both `-caption` and message text with files is an error.

### Telegram limits

| Limit | What telegram-send does |
| --- | --- |
| Message text: 4096 characters | Splits the text into several messages. |
| Caption: 1024 characters | Sends the files without a caption, then the text as a separate message. |
| Album: 2–10 items | Sends longer lists as several albums, with the caption on the first item. |
| File size: 50 MB | Nothing. Telegram rejects larger files with an error. |

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Everything was sent. |
| `1` | Something failed. The error is printed to stderr. |

When text or an album list is sent in several parts, sending stops at the first failed part and the error says which part it was, for example `part 2/3: ...`. Parts before it have already been delivered.

## Configuration

The config is read from the first `config.yaml` found in:

1. `/etc/telegram-send/`
2. `./config/`
3. the current directory

```yaml
telegram:
  token: "123456:ABC..."   # Required: bot token from @BotFather
  chatid: "123456789"      # Required: chat to send to
  threadid: 123            # Optional: forum topic ID
  api_url: "https://tg.example.com"      # Optional: Bot API mirror, defaults to https://api.telegram.org
  proxy: "socks5://user:pass@host:1080"  # Optional: http://, https:// or socks5:// proxy
```

If `proxy` is not set, the `HTTPS_PROXY` and `HTTP_PROXY` environment variables are used.

> [!WARNING]
> `config/config.yaml` in this repository is a template. Keep your real token in `/etc/telegram-send/config.yaml` so it never ends up in git.

### When api.telegram.org is blocked

There are two options. Both require a server that can reach Telegram.

**A Bot API mirror (recommended).** Point `api_url` to your own domain that proxies requests to Telegram. From outside, the traffic looks like ordinary HTTPS to your domain. A minimal nginx config:

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

**A proxy.** Set `proxy` in the config. This is simpler, but a plain HTTP or SOCKS5 proxy exposes the `api.telegram.org` hostname in the traffic, so it can be filtered by DPI.

## Building

You need Go 1.22 or newer.

```sh
git clone https://github.com/raccoon4x/telegram-send.git
cd telegram-send
go build -o telegram-send ./cmd/telegram-send
```

Static Linux binary:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o telegram-send ./cmd/telegram-send
```

The build is reproducible. The same commit, the same Go version and the same command produce a byte-identical binary. Build from a clean checkout: uncommitted or untracked files in the working tree mark the binary as modified (`vcs.modified=true` in `go version -m telegram-send`) and change its hash.

Run the tests:

```sh
go test ./...
```

---

This was my first project in Go.
