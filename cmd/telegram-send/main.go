package main

import (
	"errors"
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
	files   string
	image   string
	images  string
	caption string
)

func init() {
	flag.StringVar(&message, "message", "", "Message to send")
	flag.BoolVar(&stdin, "stdin", false, "Read message from stdin")
	flag.StringVar(&file, "file", "", "File to send")
	flag.StringVar(&files, "files", "", "Comma-separated list of files to send as document albums")
	flag.StringVar(&image, "image", "", "Image to send")
	flag.StringVar(&images, "images", "", "Comma-separated list of images to send as photo albums")
	flag.StringVar(&caption, "caption", "", "Caption for the files or images (message text is used if not set)")
}

func splitPaths(value string) []string {
	parts := strings.Split(value, ",")
	paths := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			paths = append(paths, trimmed)
		}
	}
	return paths
}

// selectFiles returns the files to send and their type ("Document" or
// "Photo"). Only one of -file, -files, -image and -images may be set.
func selectFiles(file, files, image, images string) ([]string, string, error) {
	var paths []string
	var fileType string
	set := 0

	if file != "" {
		paths, fileType = []string{file}, "Document"
		set++
	}
	if files != "" {
		paths, fileType = splitPaths(files), "Document"
		set++
	}
	if image != "" {
		paths, fileType = []string{image}, "Photo"
		set++
	}
	if images != "" {
		paths, fileType = splitPaths(images), "Photo"
		set++
	}

	if set > 1 {
		return nil, "", errors.New("only one of -file, -files, -image, -images can be used")
	}
	if set == 1 && len(paths) == 0 {
		return nil, "", errors.New("no file paths given")
	}
	return paths, fileType, nil
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadConfig("/etc/telegram-send/", "./config")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	tg := cfg.Telegram
	if tg.Token == "" || tg.ChatID == "" {
		return errors.New("token and chatid are required in the config file")
	}

	paths, fileType, err := selectFiles(file, files, image, images)
	if err != nil {
		return err
	}

	if len(flag.Args()) > 0 {
		message = strings.Join(flag.Args(), " ")
	}

	isPiped := false
	if fi, err := os.Stdin.Stat(); err == nil {
		isPiped = (fi.Mode() & os.ModeCharDevice) == 0
	}

	if message == "" && (stdin || isPiped) {
		var sb strings.Builder
		if _, err := io.Copy(&sb, os.Stdin); err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		message = sb.String()
	}

	client, err := telegram.NewClient(tg.APIURL, tg.Proxy, tg.Token, tg.ChatID, tg.ThreadID)
	if err != nil {
		return err
	}

	if len(paths) == 0 {
		if strings.TrimSpace(message) == "" {
			return errors.New("no message, file or image to send")
		}
		return client.SendMessage(message)
	}

	// With files, the message text becomes the caption.
	if strings.TrimSpace(message) != "" {
		if caption != "" {
			return errors.New("use either -caption or message text with files, not both")
		}
		caption = strings.TrimSpace(message)
	}

	// Telegram rejects long captions, so send such text as a separate message.
	if telegram.TextLength(caption) > telegram.MaxCaptionLength {
		if err := client.SendFiles(paths, "", fileType); err != nil {
			return err
		}
		if err := client.SendMessage(caption); err != nil {
			return fmt.Errorf("files were sent, but the caption text failed: %w", err)
		}
		return nil
	}

	return client.SendFiles(paths, caption, fileType)
}
