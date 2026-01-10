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
	SourceType       string `yaml:"source_type"`
	Endpoint         string `yaml:"endpoint"`
	LarkWebhookURL   string `yaml:"lark_webhook_url"`
	LarkMessageTitle string `yaml:"lark_message_title"`
}

type Configs struct {
	Configs []Config `yaml:"configs"`
}

// Handler functions for different source types
// TODO: Implement your custom logic here based on source_type

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
			if err := sendImageToLark(config.LarkWebhookURL, webhook.ImageKey); err != nil {
				log.Printf("Error forwarding image to Lark: %v", err)
				// Note: We still return 200 even if Lark forwarding fails
				// This prevents generic webhook from retrying the webhook
			}
		} else {
			if err := sendToLark(config.LarkWebhookURL, webhook.Message, webhook.Title); err != nil {
				log.Printf("Error forwarding to Lark: %v", err)
				// Note: We still return 200 even if Lark forwarding fails
				// This prevents generic webhook from retrying the webhook
			}
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
		fmt.Printf("  Lark Webhook Endpoint: %s\n", config.LarkWebhookURL)
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

