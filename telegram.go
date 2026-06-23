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
	// Log the start of the operation
	log.Printf("[Telegram] Starting to send message (rawMode: %v)", rawMode)

	// Validate configuration
	if botToken == "" {
		log.Printf("[Telegram] ERROR: Bot token is empty")
		return fmt.Errorf("telegram bot token is empty")
	}
	if chatID == "" {
		log.Printf("[Telegram] ERROR: Chat ID is empty")
		return fmt.Errorf("telegram chat ID is empty")
	}

	// Log configuration (masking the bot token for security)
	maskedToken := ""
	if len(botToken) > 8 {
		maskedToken = botToken[:4] + "..." + botToken[len(botToken)-4:]
	} else {
		maskedToken = "***"
	}
	log.Printf("[Telegram] Config - Bot Token: %s, Chat ID: %s, Chat Sub ID: %s", maskedToken, chatID, chatSubID)

	// Build the Telegram API URL
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	log.Printf("[Telegram] API URL: https://api.telegram.org/bot%s/sendMessage", maskedToken)

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
		log.Printf("[Telegram] Using Markdown parse mode")
	} else {
		log.Printf("[Telegram] Using raw text mode (no Markdown parsing)")
	}

	// Add message_thread_id if chat_sub_id is provided (for topics in groups)
	if chatSubID != "" {
		telegramMsg.MessageThreadID = chatSubID
		log.Printf("[Telegram] Using message thread ID: %s", chatSubID)
	}

	// Marshal the message to JSON
	payload, err := json.Marshal(telegramMsg)
	if err != nil {
		log.Printf("[Telegram] ERROR: Failed to marshal message to JSON: %v", err)
		return err
	}

	// Log the payload (truncate if too long)
	payloadStr := string(payload)
	if len(payloadStr) > 500 {
		log.Printf("[Telegram] Request payload (truncated): %s...", payloadStr[:500])
	} else {
		log.Printf("[Telegram] Request payload: %s", payloadStr)
	}

	// Send to Telegram API
	log.Printf("[Telegram] Sending POST request to Telegram API...")
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("[Telegram] ERROR: HTTP request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	log.Printf("[Telegram] Received response with status code: %d", resp.StatusCode)

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[Telegram] ERROR: Failed to read response body: %v", err)
		return err
	}

	log.Printf("[Telegram] Response body: %s", string(bodyBytes))

	// Parse response
	var telegramResp TelegramResponse
	if err := json.Unmarshal(bodyBytes, &telegramResp); err != nil {
		log.Printf("[Telegram] ERROR: Failed to parse response JSON: %v", err)
		log.Printf("[Telegram] Raw response body was: %s", string(bodyBytes))
		return err
	}

	// Log detailed response information
	log.Printf("[Telegram] API Response - Status Code: %d, OK: %v, Description: %s",
		resp.StatusCode, telegramResp.OK, telegramResp.Description)

	// Check response status
	if !telegramResp.OK {
		log.Printf("[Telegram] ERROR: Telegram API returned error: %s", telegramResp.Description)
		return fmt.Errorf("telegram API error: %s", telegramResp.Description)
	}

	log.Printf("[Telegram] SUCCESS: Message sent successfully to chat %s", chatID)
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
