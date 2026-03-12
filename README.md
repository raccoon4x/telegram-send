# Telegram Send

This is my first project using GoLang. This project provides a simple command-line tool to send messages, files, and images to a Telegram chat. It is built in Go and utilizes the Telegram Bot API for communication.

## Features

- Send text messages to a specified Telegram chat.
- Send files and images with optional captions.
- Read messages from stdin to send to a Telegram chat.

## Usage

### Send a message

- With args:

  telegram-send "hello world"

- With stdin (pipe, no flags needed):

  cat /var/log/log.txt | grep "hallo" | telegram-send

- Force reading stdin (even if you're not piping):

  telegram-send -stdin

## Prerequisites

Before you can use this tool, you need to:

1. Create a Telegram bot by talking to [@BotFather](https://t.me/botfather) and get the bot token.
2. Find out the chat ID where you want to send messages.
