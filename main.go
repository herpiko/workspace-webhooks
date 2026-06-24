package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gopkg.in/yaml.v3"
)

type Config struct {
	SourceType        string   `yaml:"source_type"`
	Endpoint          string   `yaml:"endpoint"`
	LarkWebhookURL    string   `yaml:"lark_webhook_url"`
	LarkMessageTitle  string   `yaml:"lark_message_title"`
	TelegramBotToken  string   `yaml:"telegram_bot_token"`
	TelegramChatID    string   `yaml:"telegram_chat_id"`
	TelegramChatSubID string   `yaml:"telegram_chat_sub_id"`
	GitLabBranches    []string `yaml:"gitlab_branches"`     // Branches to monitor, empty = all branches
	GitLabStatuses    []string `yaml:"gitlab_statuses"`     // Pipeline/build statuses to notify, empty = failed only
	GitLabNotifyAll   bool     `yaml:"gitlab_notify_all"`   // If true, notify all pipeline/build events regardless of status
}

type Configs struct {
	AllowedEvents []string `yaml:"allowed_events"`
	Configs       []Config `yaml:"configs"`
}

// Global allowed events list
var allowedEvents []string

// isEventAllowed checks if an event type is allowed to be processed
// If allowedEvents is empty, all events are allowed
func isEventAllowed(eventType string) bool {
	if len(allowedEvents) == 0 {
		return true
	}
	for _, allowed := range allowedEvents {
		if allowed == eventType {
			return true
		}
	}
	log.Printf("Event type '%s' is not in allowed events list, skipping", eventType)
	return false
}

// Handler functions for different source types

// sendNotification sends message to configured notification channels (Lark and/or Telegram)
func sendNotification(config Config, message string, title string) {
	// Send to Lark if configured
	if config.LarkWebhookURL != "" {
		log.Printf("[Notification] Sending to Lark...")
		if err := sendToLark(config.LarkWebhookURL, message, title); err != nil {
			log.Printf("[Notification] Error forwarding to Lark: %v", err)
		}
	}

	// Send to Telegram if configured
	if config.TelegramBotToken != "" && config.TelegramChatID != "" {
		log.Printf("[Notification] Sending to Telegram...")
		if err := sendToTelegram(config.TelegramBotToken, config.TelegramChatID, config.TelegramChatSubID, message, title); err != nil {
			log.Printf("[Notification] Error forwarding to Telegram: %v", err)
		}
	} else {
		if config.TelegramBotToken == "" && config.TelegramChatID == "" {
			log.Printf("[Notification] Telegram not configured (both bot token and chat ID are empty)")
		} else if config.TelegramBotToken == "" {
			log.Printf("[Notification] Telegram not configured (bot token is empty)")
		} else if config.TelegramChatID == "" {
			log.Printf("[Notification] Telegram not configured (chat ID is empty)")
		}
	}
}

// sendNotificationRaw sends message without Markdown parsing (for user-provided content)
func sendNotificationRaw(config Config, message string, title string) {
	// Send to Lark if configured
	if config.LarkWebhookURL != "" {
		log.Printf("[Notification] Sending to Lark (raw mode)...")
		if err := sendToLark(config.LarkWebhookURL, message, title); err != nil {
			log.Printf("[Notification] Error forwarding to Lark: %v", err)
		}
	}

	// Send to Telegram if configured (raw mode to avoid Markdown parsing errors)
	if config.TelegramBotToken != "" && config.TelegramChatID != "" {
		log.Printf("[Notification] Sending to Telegram (raw mode)...")
		if err := sendToTelegramRaw(config.TelegramBotToken, config.TelegramChatID, config.TelegramChatSubID, message, title); err != nil {
			log.Printf("[Notification] Error forwarding to Telegram: %v", err)
		}
	} else {
		if config.TelegramBotToken == "" && config.TelegramChatID == "" {
			log.Printf("[Notification] Telegram not configured (both bot token and chat ID are empty)")
		} else if config.TelegramBotToken == "" {
			log.Printf("[Notification] Telegram not configured (bot token is empty)")
		} else if config.TelegramChatID == "" {
			log.Printf("[Notification] Telegram not configured (chat ID is empty)")
		}
	}
}

// sendImageNotification sends image to configured notification channels
func sendImageNotification(config Config, imageKey string) {
	// Send to Lark if configured (Telegram doesn't support image_key format)
	if config.LarkWebhookURL != "" {
		if err := sendImageToLark(config.LarkWebhookURL, imageKey); err != nil {
			log.Printf("Error forwarding image to Lark: %v", err)
		}
	}
}

// GitHub webhook handler is now implemented in github.go

type GenericWebhook struct {
	Title    string `json:"title"`
	Message  string `json:"message"`
	ImageKey string `json:"image_key"`
}

// Generic handler for unknown source types
func handleGenericWebhook(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received generic webhook request")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading generic webhook body: %v", err)
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Parse the generic webhook payload
		var webhook GenericWebhook
		if err := json.Unmarshal(body, &webhook); err != nil {
			log.Printf("Error parsing generic webhook JSON: %v", err)
			http.Error(w, "Error parsing JSON", http.StatusBadRequest)
			return
		}

		// Check if image_key is provided, if so send image message
		if webhook.ImageKey != "" {
			sendImageNotification(config, webhook.ImageKey)
		} else {
			// Use raw mode for generic webhooks to avoid Markdown parsing errors
			// since user-provided content may contain special characters
			sendNotificationRaw(config, webhook.Message, webhook.Title)
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Generic webhook received\n")
	}
}

// getHandlerForSourceType returns the appropriate handler based on source type
func getHandlerForSourceType(config Config) http.HandlerFunc {
	switch config.SourceType {
	case "github":
		return handleGitHubWebhook(config)
	case "gitlab":
		return handleGitLabWebhook(config)
	case "grafana":
		return handleGrafanaWebhook(config)
	case "generic":
		return handleGenericWebhook(config)
	default:
		return handleGenericWebhook(config)
	}
}

func main() {
	// Define command-line flags
	configPath := flag.String("c", "", "Path to the configuration file (required)")
	portShort := flag.String("p", "8080", "Port to run the server on")
	portLong := flag.String("port", "", "Port to run the server on (alias for -p)")
	flag.Parse()

	// Determine which port flag was used, prioritizing --port if both are set
	port := *portShort
	if *portLong != "" {
		port = *portLong
	}

	// Check if config path is provided
	if *configPath == "" {
		log.Fatal("Error: configuration file path is required. Use -c flag to specify the path.")
	}

	// Read the YAML file
	data, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	// Parse the YAML
	var configs Configs
	err = yaml.Unmarshal(data, &configs)
	if err != nil {
		log.Fatalf("Error parsing YAML: %v", err)
	}

	// Set global allowed events
	allowedEvents = configs.AllowedEvents
	if len(allowedEvents) > 0 {
		log.Printf("Allowed events configured: %v", allowedEvents)
	} else {
		log.Printf("No allowed events configured - all events will be processed")
	}

	// Create chi router
	r := chi.NewRouter()

	// Add middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Display the parsed configuration and setup routes
	fmt.Printf("Successfully loaded %d configuration(s):\n\n", len(configs.Configs))

	// Setup dynamic routes based on config
	for i, config := range configs.Configs {
		fmt.Printf("Config %d:\n", i+1)
		fmt.Printf("  Source Type: %s\n", config.SourceType)
		fmt.Printf("  Endpoint: %s\n", config.Endpoint)
		if config.LarkWebhookURL != "" {
			fmt.Printf("  Lark Webhook Endpoint: %s\n", config.LarkWebhookURL)
		}
		if config.TelegramBotToken != "" && config.TelegramChatID != "" {
			fmt.Printf("  Telegram Chat ID: %s\n", config.TelegramChatID)
			if config.TelegramChatSubID != "" {
				fmt.Printf("  Telegram Chat Sub ID: %s\n", config.TelegramChatSubID)
			}
		}
		fmt.Println()

		// Register route dynamically with /webhook prefix
		handler := getHandlerForSourceType(config)
		routePath := "/webhook" + config.Endpoint
		r.Post(routePath, handler)
		log.Printf("Registered POST route: %s -> %s handler", routePath, config.SourceType)
	}

	// Start HTTP server
	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("\nStarting server on %s\n", addr)
	log.Printf("Server listening on %s", addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

