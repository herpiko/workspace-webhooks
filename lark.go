package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// LarkCardMessage represents a Lark card message payload
type LarkCardMessage struct {
	MsgType string      `json:"msg_type"`
	Content LarkContent `json:"content"`
}

// LarkContent contains the card content
type LarkContent struct {
	Post LarkPost `json:"post"`
}

// LarkPost contains the post content with language and title
type LarkPost struct {
	EnUS LarkPostContent `json:"en_us"`
}

// LarkPostContent contains the actual post content
type LarkPostContent struct {
	Title   string             `json:"title"`
	Content [][]LarkTextBlock  `json:"content"`
}

// LarkTextBlock represents a text block in the post
type LarkTextBlock struct {
	Tag  string `json:"tag"`
	Text string `json:"text"`
}

// LarkImageMessage represents a Lark image message payload
type LarkImageMessage struct {
	MsgType string           `json:"msg_type"`
	Content LarkImageContent `json:"content"`
}

// LarkImageContent contains the image_key
type LarkImageContent struct {
	ImageKey string `json:"image_key"`
}

// sendImageToLark sends an image message to Lark webhook endpoint
func sendImageToLark(larkWebhookURL string, imageKey string) error {
	// Create Lark image message
	larkMsg := LarkImageMessage{
		MsgType: "image",
		Content: LarkImageContent{
			ImageKey: imageKey,
		},
	}

	// Marshal the message to JSON
	payload, err := json.Marshal(larkMsg)
	if err != nil {
		log.Printf("Error marshaling Lark image message: %v", err)
		return err
	}

	// Send to Lark webhook
	resp, err := http.Post(larkWebhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Error sending image message to Lark: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading Lark response body: %v", err)
		return err
	}

	// Print response
	log.Printf("Lark API Response (Status: %d): %s", resp.StatusCode, string(bodyBytes))

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("lark webhook returned status %d", resp.StatusCode)
	}

	log.Printf("Successfully sent image message to Lark")
	return nil
}

// sendToLark sends the formatted message to Lark webhook endpoint
func sendToLark(larkWebhookURL string, message string, title string) error {
	// Create Lark card message with the formatted text
	larkMsg := LarkCardMessage{
		MsgType: "post",
		Content: LarkContent{
			Post: LarkPost{
				EnUS: LarkPostContent{
					Title: title,
					Content: [][]LarkTextBlock{
						{
							{
								Tag:  "text",
								Text: message,
							},
						},
					},
				},
			},
		},
	}

	if len(title) == 0 {
		larkMsg = LarkCardMessage{
			MsgType: "post",
			Content: LarkContent{
				Post: LarkPost{
					EnUS: LarkPostContent{
						Content: [][]LarkTextBlock{
							{
								{
									Tag:  "text",
									Text: message,
								},
							},
						},
					},
				},
			},
		}
	}

	// Marshal the message to JSON
	payload, err := json.Marshal(larkMsg)
	if err != nil {
		log.Printf("Error marshaling Lark message: %v", err)
		return err
	}

	// Send to Lark webhook
	resp, err := http.Post(larkWebhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Error sending message to Lark: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading Lark response body: %v", err)
		return err
	}

	// Print response
	log.Printf("Lark API Response (Status: %d): %s", resp.StatusCode, string(bodyBytes))

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("lark webhook returned status %d", resp.StatusCode)
	}

	log.Printf("Successfully sent message to Lark")
	return nil
}
