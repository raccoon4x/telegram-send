package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func sendMessage(token, chatID, text string, threadID int) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}
	if threadID > 0 {
		payload["message_thread_id"] = threadID
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send message: %s", resp.Status)
	}

	return nil
}

func sendFile(token, chatID, filePath, caption string, fileType string, threadID int) error {
	var url string
	switch fileType {
	case "Document":
		url = fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", token)
	case "Photo":
		url = fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", token)
	default:
		return fmt.Errorf("unsupported file type: %s", fileType)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fieldName := "document"
	if fileType == "Photo" {
		fieldName = "photo"
	}
	fw, err := w.CreateFormFile(fieldName, filepath.Base(file.Name()))
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, file); err != nil {
		return err
	}
	w.WriteField("chat_id", chatID)
	if threadID > 0 {
		w.WriteField("message_thread_id", fmt.Sprintf("%d", threadID))
	}
	if caption != "" {
		w.WriteField("caption", caption)
	}
	w.Close()

	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send file: %s", resp.Status)
	}

	return nil
}

func SendMessage(token, chatID, text string, threadID int) error {
	return sendMessage(token, chatID, text, threadID)
}

func SendFile(token, chatID, filePath, caption, fileType string, threadID int) error {
	return sendFile(token, chatID, filePath, caption, fileType, threadID)
}
