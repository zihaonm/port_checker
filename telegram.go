package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TelegramNotifier handles sending notifications to Telegram
type TelegramNotifier struct {
	BotToken string
	ChatID   string
}

// telegramMessage represents the message structure for Telegram API
type telegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// NewTelegramNotifier creates a new TelegramNotifier instance
func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		BotToken: botToken,
		ChatID:   chatID,
	}
}

// SendDownAlert sends a notification when an endpoint is down
func (t *TelegramNotifier) SendDownAlert(endpoint string) error {
	message := fmt.Sprintf("⚠️ [DOWN] %s is not reachable", endpoint)
	return t.sendMessage(message)
}

// SendUpAlert sends a notification when an endpoint is back up
func (t *TelegramNotifier) SendUpAlert(endpoint string) error {
	message := fmt.Sprintf("✅ [UP] %s is now reachable", endpoint)
	return t.sendMessage(message)
}

// sendMessage sends a message to Telegram
func (t *TelegramNotifier) sendMessage(text string) error {
	if t.BotToken == "" || t.ChatID == "" {
		return fmt.Errorf("telegram bot token or chat ID not configured")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.BotToken)

	msg := telegramMessage{
		ChatID: t.ChatID,
		Text:   text,
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status: %d", resp.StatusCode)
	}

	return nil
}
