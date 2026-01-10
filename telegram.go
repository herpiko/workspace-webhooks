package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// TelegramMessage represents a Telegram message payload
type TelegramMessage struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	MessageThreadID       string `json:"message_thread_id,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
}

// TelegramResponse represents the response from Telegram API
type TelegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// sendToTelegram sends the formatted message to Telegram using Bot API
func sendToTelegram(botToken string, chatID string, chatSubID string, message string, title string) error {
	return sendToTelegramWithOptions(botToken, chatID, chatSubID, message, title, false)
}

// sendToTelegramRaw sends the message to Telegram without Markdown parsing
func sendToTelegramRaw(botToken string, chatID string, chatSubID string, message string, title string) error {
	return sendToTelegramWithOptions(botToken, chatID, chatSubID, message, title, true)
}

// sendToTelegramWithOptions sends the formatted message to Telegram using Bot API
func sendToTelegramWithOptions(botToken string, chatID string, chatSubID string, message string, title string, rawMode bool) error {
	// Build the Telegram API URL
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	// Use message directly without title for compact display
	formattedMessage := message

	// Create Telegram message
	telegramMsg := TelegramMessage{
		ChatID:                chatID,
		Text:                  formattedMessage,
		DisableWebPagePreview: true,
	}

	// Only use Markdown parse mode if not in raw mode
	if !rawMode {
		telegramMsg.ParseMode = "Markdown"
	}

	// Add message_thread_id if chat_sub_id is provided (for topics in groups)
	if chatSubID != "" {
		telegramMsg.MessageThreadID = chatSubID
	}

	// Marshal the message to JSON
	payload, err := json.Marshal(telegramMsg)
	if err != nil {
		log.Printf("Error marshaling Telegram message: %v", err)
		return err
	}

	// Send to Telegram API
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Error sending message to Telegram: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading Telegram response body: %v", err)
		return err
	}

	// Parse response
	var telegramResp TelegramResponse
	if err := json.Unmarshal(bodyBytes, &telegramResp); err != nil {
		log.Printf("Error parsing Telegram response: %v", err)
		return err
	}

	// Print response
	log.Printf("Telegram API Response (Status: %d, OK: %v): %s", resp.StatusCode, telegramResp.OK, string(bodyBytes))

	// Check response status
	if !telegramResp.OK {
		return fmt.Errorf("telegram API error: %s", telegramResp.Description)
	}

	log.Printf("Successfully sent message to Telegram")
	return nil
}

// escapeMarkdown escapes special characters for Telegram Markdown
func escapeMarkdown(text string) string {
	// In Markdown mode, we only need to escape these characters in the title
	// The message body is left as-is since it may contain intentional formatting
	replacer := []string{
		"_", "\\_",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	}

	result := text
	for i := 0; i < len(replacer); i += 2 {
		result = replaceAll(result, replacer[i], replacer[i+1])
	}
	return result
}

// replaceAll replaces all occurrences of old with new in s
func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old) - 1
		} else {
			result += string(s[i])
		}
	}
	return result
}
