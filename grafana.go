package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

// GrafanaAlert represents a single alert in the Grafana webhook
type GrafanaAlert struct {
	Status       string                 `json:"status"`
	Labels       map[string]string      `json:"labels"`
	Annotations  map[string]string      `json:"annotations"`
	StartsAt     string                 `json:"startsAt"`
	EndsAt       string                 `json:"endsAt"`
	GeneratorURL string                 `json:"generatorURL"`
	Fingerprint  string                 `json:"fingerprint"`
	SilenceURL   string                 `json:"silenceURL"`
	DashboardURL string                 `json:"dashboardURL"`
	PanelURL     string                 `json:"panelURL"`
	Values       map[string]interface{} `json:"values"`
}

// GrafanaWebhook represents the complete Grafana webhook payload
type GrafanaWebhook struct {
	Receiver          string            `json:"receiver"`
	Status            string            `json:"status"`
	OrgID             int               `json:"orgId"`
	Alerts            []GrafanaAlert    `json:"alerts"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	Title             string            `json:"title"`
	State             string            `json:"state"`
	Message           string            `json:"message"`
}

// handleGrafanaWebhook handles incoming Grafana webhook requests
func handleGrafanaWebhook(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received Grafana webhook request")

		// Read the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading Grafana webhook body: %v", err)
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Convert body to string and print for inspection
		bodyString := string(body)
		log.Printf("Grafana webhook raw payload:\n %s", bodyString)

		// Parse the Grafana webhook payload
		var webhook GrafanaWebhook
		if err := json.Unmarshal([]byte(bodyString), &webhook); err != nil {
			log.Printf("Error parsing Grafana webhook JSON: %v", err)
			http.Error(w, "Error parsing JSON", http.StatusBadRequest)
			return
		}

		log.Printf("Grafana webhook - Status: %s, Alerts: %d, Title: %s",
			webhook.Status, len(webhook.Alerts), webhook.Title)

		ignoreAlert := false

		// Construct formatted message from alerts
		var message string
		title := config.LarkMessageTitle
		for _, alert := range webhook.Alerts {
			status := alert.Status
			if status == "resolved" {
				status += " ✅"
			} else if status == "firing" {
				status += " 🚨"
			} else {
				status += " ⚠️"
			}
			message += "Status: " + status + "\n"

			if alertname, ok := alert.Labels["alertname"]; ok {
				title = alertname
			}

			if title == "DatasourceNoData" {
				ignoreAlert = true
			}

			if namespace, ok := alert.Labels["namespace"]; ok {
				message += "Namespace: " + namespace + "\n"
			}

			if service, ok := alert.Labels["service"]; ok {
				message += "Service: " + service + "\n"
			}

			if game, ok := alert.Labels["game"]; ok {
				message += "Game: " + game + "\n"
			}

			if pod, ok := alert.Labels["pod"]; ok {
				message += "Pod / Instance: " + pod + "\n"
			}

			// Get summary from annotations
			if summary, ok := alert.Annotations["summary"]; ok {
				message += "Summary: " + summary + "\n"
			} else {
				message += "Summary: \n"
			}

			message += "Alert Start at: " + alert.StartsAt + "\n"
			message += "Source: " + alert.GeneratorURL + "\n"
			message += "\n"
		}

		if !ignoreAlert {
			// Send message to Lark webhook endpoint with "Grafana" title
			if err := sendToLark(config.LarkWebhookURL, message, title); err != nil {
				log.Printf("Error forwarding to Lark: %v", err)
				// Note: We still return 200 even if Lark forwarding fails
				// This prevents Grafana from retrying the webhook
			}
		}

		// Return empty payload with 200 status code
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}
}
