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
func (t *TelegramNotifier) SendDownAlert(endpoint string, name string, failureCount int, errorMsg string) error {
	var message string
	displayName := endpoint
	if name != "" {
		displayName = fmt.Sprintf("%s (%s)", endpoint, name)
	}

	if failureCount > 1 {
		message = fmt.Sprintf("🚨 [DOWN] %s is not reachable\n\nFailure count: %d\nError: %s", displayName, failureCount, errorMsg)
	} else {
		message = fmt.Sprintf("⚠️ [DOWN] %s is not reachable\n\nError: %s", displayName, errorMsg)
	}
	return t.sendMessage(message)
}

// SendUpAlert sends a notification when an endpoint is back up
func (t *TelegramNotifier) SendUpAlert(endpoint string, name string, failureCount int, downtime time.Duration) error {
	displayName := endpoint
	if name != "" {
		displayName = fmt.Sprintf("%s (%s)", endpoint, name)
	}

	downtimeStr := formatDuration(downtime)
	message := fmt.Sprintf("✅ [UP] %s is now reachable\n\nWas down for: %s\nFailed checks: %d", displayName, downtimeStr, failureCount)
	return t.sendMessage(message)
}

// SendCertExpiryWarning sends a warning about expiring SSL certificate
func (t *TelegramNotifier) SendCertExpiryWarning(endpoint string, name string, expiryDate time.Time) error {
	displayName := endpoint
	if name != "" {
		displayName = fmt.Sprintf("%s (%s)", endpoint, name)
	}

	daysUntilExpiry := time.Until(expiryDate).Hours() / 24
	message := fmt.Sprintf("⚠️ [SSL WARNING] %s\n\nSSL certificate expires in %.0f days\nExpiry date: %s",
		displayName, daysUntilExpiry, expiryDate.Format("2006-01-02 15:04"))
	return t.sendMessage(message)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0f minutes", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1f hours", d.Hours())
	}
	return fmt.Sprintf("%.1f days", d.Hours()/24)
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
