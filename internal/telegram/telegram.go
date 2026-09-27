package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultAPIURL = "https://api.telegram.org"

	// Telegram limits, counted in UTF-16 code units.
	MaxMessageLength = 4096
	MaxCaptionLength = 1024

	maxMediaGroupSize = 10
	requestTimeout    = 15 * time.Second
)

type Client struct {
	apiURL   string
	token    string
	chatID   string
	threadID int
	http     *http.Client
}

// NewClient creates a Telegram Bot API client. An empty apiURL means the
// official API. An empty proxy means the proxy is taken from the
// HTTPS_PROXY/HTTP_PROXY environment variables.
func NewClient(apiURL, proxy, token, chatID string, threadID int) (*Client, error) {
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy %q: %w", proxy, err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &Client{
		apiURL:   strings.TrimRight(apiURL, "/"),
		token:    token,
		chatID:   chatID,
		threadID: threadID,
		http:     &http.Client{Timeout: requestTimeout, Transport: transport},
	}, nil
}

// TextLength returns the length of s as Telegram counts it.
func TextLength(s string) int {
	n := 0
	for _, r := range s {
		n += utf16Len(r)
	}
	return n
}

func utf16Len(r rune) int {
	if r >= 0x10000 {
		return 2
	}
	return 1
}

// splitText splits text into chunks of at most limit UTF-16 code units,
// preferring to cut at line breaks. Blank chunks are dropped.
func splitText(text string, limit int) []string {
	var chunks []string
	add := func(chunk string) {
		if strings.TrimSpace(chunk) != "" {
			chunks = append(chunks, chunk)
		}
	}

	for TextLength(text) > limit {
		cut, size := 0, 0
		for i, r := range text {
			size += utf16Len(r)
			if size > limit {
				cut = i
				break
			}
		}
		if nl := strings.LastIndexByte(text[:cut], '\n'); nl > 0 {
			add(text[:nl])
			text = text[nl+1:]
		} else {
			add(text[:cut])
			text = text[cut:]
		}
	}
	add(text)
	return chunks
}

// SendMessage sends text, splitting it into several messages if it is longer
// than MaxMessageLength.
func (c *Client) SendMessage(text string) error {
	chunks := splitText(text, MaxMessageLength)
	for i, chunk := range chunks {
		if err := c.sendMessage(chunk); err != nil {
			if len(chunks) > 1 {
				return fmt.Errorf("part %d/%d: %w", i+1, len(chunks), err)
			}
			return err
		}
	}
	return nil
}

func (c *Client) sendMessage(text string) error {
	payload := map[string]interface{}{
		"chat_id": c.chatID,
		"text":    text,
	}
	if c.threadID > 0 {
		payload["message_thread_id"] = c.threadID
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.methodURL("sendMessage"), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	return c.do(req)
}

// SendFiles sends files of the given type ("Document" or "Photo"). A single
// file is sent as is, several files are sent as albums of up to 10 items.
// The caption is attached to the first file only.
func (c *Client) SendFiles(filePaths []string, caption, fileType string) error {
	if fileType != "Document" && fileType != "Photo" {
		return fmt.Errorf("unsupported file type: %s", fileType)
	}
	if len(filePaths) == 0 {
		return fmt.Errorf("no files provided")
	}

	var batches [][]string
	for start := 0; start < len(filePaths); start += maxMediaGroupSize {
		end := min(start+maxMediaGroupSize, len(filePaths))
		batches = append(batches, filePaths[start:end])
	}

	for i, batch := range batches {
		batchCaption := ""
		if i == 0 {
			batchCaption = caption
		}

		var err error
		if len(batch) == 1 {
			err = c.sendFile(batch[0], batchCaption, fileType)
		} else {
			err = c.sendMediaGroup(batch, batchCaption, fileType)
		}
		if err != nil {
			if len(batches) > 1 {
				return fmt.Errorf("album %d/%d: %w", i+1, len(batches), err)
			}
			return err
		}
	}
	return nil
}

func (c *Client) sendFile(filePath, caption, fileType string) error {
	method, fieldName := "sendDocument", "document"
	if fileType == "Photo" {
		method, fieldName = "sendPhoto", "photo"
	}

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	if err := attachFile(w, fieldName, filePath); err != nil {
		return err
	}
	if err := c.writeCommonFields(w); err != nil {
		return err
	}
	if caption != "" {
		if err := w.WriteField("caption", caption); err != nil {
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.methodURL(method), &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	return c.do(req)
}

func (c *Client) sendMediaGroup(filePaths []string, caption, fileType string) error {
	mediaType := "document"
	if fileType == "Photo" {
		mediaType = "photo"
	}

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	media := make([]map[string]string, 0, len(filePaths))
	for i, filePath := range filePaths {
		partName := fmt.Sprintf("file%d", i)
		if err := attachFile(w, partName, filePath); err != nil {
			return err
		}

		item := map[string]string{
			"type":  mediaType,
			"media": "attach://" + partName,
		}
		if i == 0 && caption != "" {
			item["caption"] = caption
		}
		media = append(media, item)
	}

	mediaJSON, err := json.Marshal(media)
	if err != nil {
		return err
	}
	if err := c.writeCommonFields(w); err != nil {
		return err
	}
	if err := w.WriteField("media", string(mediaJSON)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.methodURL("sendMediaGroup"), &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	return c.do(req)
}

func attachFile(w *multipart.Writer, fieldName, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	part, err := w.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)
	return err
}

func (c *Client) writeCommonFields(w *multipart.Writer) error {
	if err := w.WriteField("chat_id", c.chatID); err != nil {
		return err
	}
	if c.threadID > 0 {
		if err := w.WriteField("message_thread_id", fmt.Sprintf("%d", c.threadID)); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) methodURL(method string) string {
	return fmt.Sprintf("%s/bot%s/%s", c.apiURL, c.token, method)
}

// do sends the request and turns a failed Bot API response into an error
// with Telegram's description. The URL is not included in errors because it
// contains the token.
func (c *Client) do(req *http.Request) error {
	method := req.URL.Path[strings.LastIndexByte(req.URL.Path, '/')+1:]

	resp, err := c.http.Do(req)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok {
			err = urlErr.Err
		}
		return fmt.Errorf("%s: %w", method, err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s: reading response: %w", method, err)
	}
	jsonErr := json.Unmarshal(body, &result)
	if resp.StatusCode == http.StatusOK && jsonErr == nil && result.OK {
		return nil
	}

	if result.Description != "" {
		return fmt.Errorf("%s: %s (%s)", method, result.Description, resp.Status)
	}
	return fmt.Errorf("%s: unexpected response (%s)", method, resp.Status)
}
