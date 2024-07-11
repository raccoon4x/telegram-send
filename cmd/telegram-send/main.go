package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"telegram-send/internal/config"
	"telegram-send/internal/telegram"
)

var (
	message string
	stdin   bool
	file    string
	image   string
	caption string
)

func init() {
	flag.StringVar(&message, "message", "", "Message to send")
	flag.BoolVar(&stdin, "stdin", false, "Read message from stdin")
	flag.StringVar(&file, "file", "", "File to send")
	flag.StringVar(&image, "image", "", "Image to send")
	flag.StringVar(&caption, "caption", "", "Caption for the file or image")
}

func main() {
	flag.Parse()

	cfg, err := config.LoadConfig("/etc/telegram-send/", "./config")
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	token := cfg.Telegram.Token
	chatID := cfg.Telegram.ChatID

	if token == "" || chatID == "" {
		fmt.Println("token and chatid are required in the config file")
		return
	}

	if stdin {
		var sb strings.Builder
		io.Copy(&sb, os.Stdin)
		message = sb.String()
	}

	if message != "" {
		if err := telegram.SendMessage(token, chatID, message); err != nil {
			fmt.Println("Error sending message:", err)
		}
	} else if file != "" {
		if err := telegram.SendFile(token, chatID, file, caption, "Document"); err != nil {
			fmt.Println("Error sending file:", err)
		}
	} else if image != "" {
		if err := telegram.SendFile(token, chatID, image, caption, "Photo"); err != nil {
			fmt.Println("Error sending image:", err)
		}
	} else {
		fmt.Println("No message, file or image to send")
	}
}
