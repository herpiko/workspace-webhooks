package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// GitLabEvent represents a GitLab webhook event
type GitLabEvent struct {
	ObjectKind   string `json:"object_kind"`
	UserUsername string `json:"user_username"`
	Ref          string `json:"ref"`
	Project      struct {
		WebURL        string `json:"web_url"`
		Name          string `json:"name"`
		DefaultBranch string `json:"default_branch"`
	} `json:"project"`
	ObjectAttributes struct {
		URL    string `json:"url"`
		Title  string `json:"title"`
		Action string `json:"action"`
		Status string `json:"status"` // For pipeline status
		Ref    string `json:"ref"`    // For pipeline branch
	} `json:"object_attributes"`
	User struct {
		Username string `json:"username"`
	} `json:"user"`
	TotalCommitCount   int    `json:"total_commits_count"`
	BuildName          string `json:"build_name"`
	BuildStatus        string `json:"build_status"`
	BuildFailureReason string `json:"build_failure_reason"`
}

// generateGitLabMessage generates a formatted message from GitLab event
func generateGitLabMessage(event GitLabEvent) string {
	// Check if this event type is allowed based on object_kind
	eventType := fmt.Sprintf("gitlab.%s", event.ObjectKind)
	if !isEventAllowed(eventType) {
		return ""
	}

	switch event.ObjectKind {
	case "push":
		if event.TotalCommitCount > 0 {
			// Check if push is to main or master branch
			if event.Ref == "refs/heads/main" || event.Ref == "refs/heads/master" {
				return fmt.Sprintf("🔨 New push by %s to %s (%s)\n%s/-/commits/%s",
					event.UserUsername, event.Project.Name, event.Ref, event.Project.WebURL, event.Ref)
			}
			log.Printf("GitLab push event - skipping notification (not main/master branch): %s", event.Ref)
		}
	case "pipeline":
		// Handle pipeline events
		pipelineRef := event.ObjectAttributes.Ref
		pipelineStatus := event.ObjectAttributes.Status

		// Only notify for pipelines on main/master branch
		if pipelineRef == "main" || pipelineRef == "master" {
			switch pipelineStatus {
			case "failed":
				return fmt.Sprintf("❌ Pipeline FAILED on %s/%s\n%s",
					event.Project.Name, pipelineRef, event.ObjectAttributes.URL)
			case "success":
				return fmt.Sprintf("✅ Pipeline succeeded on %s/%s\n%s",
					event.Project.Name, pipelineRef, event.ObjectAttributes.URL)
			default:
				log.Printf("GitLab pipeline event - skipping notification (status: %s, ref: %s)", pipelineStatus, pipelineRef)
			}
		} else {
			log.Printf("GitLab pipeline event - skipping notification (not main/master branch): %s (status: %s)", pipelineRef, pipelineStatus)
		}
	case "merge_request":
		/*
			if event.ObjectAttributes.Action == "approved" {
				return fmt.Sprintf("👍 MR get APPROVED by %s: %s : %s",
					event.User.Username, event.ObjectAttributes.Title, event.ObjectAttributes.URL)
					} else if event.ObjectAttributes.Action == "unapproved" {
						return fmt.Sprintf("👎 MR get UNAPPROVED by %s: %s - %s",
							event.User.Username, event.ObjectAttributes.Title, event.ObjectAttributes.URL)
			} else
		*/
		if event.ObjectAttributes.Action == "open" || event.ObjectAttributes.Action == "reopen" {
			return fmt.Sprintf("🔥 New MR opened by %s: %s - %s",
				event.User.Username, event.ObjectAttributes.Title, event.ObjectAttributes.URL)
			/*
				} else if event.ObjectAttributes.Action == "close" {
					return fmt.Sprintf("❌ Merge request get closed by %s: %s\n%s",
						event.User.Username, event.ObjectAttributes.Title, event.ObjectAttributes.URL)
			*/
		}
		/*
			} else if event.ObjectAttributes.Action == "merge" {
				return fmt.Sprintf("🎉 Merge request get MERGED by %s: %s\n%s",
					event.User.Username, event.ObjectAttributes.Title, event.ObjectAttributes.URL)
			}
				case "note":
					return fmt.Sprintf("💬 New comment by %s: \n%s",
						event.User.Username, event.ObjectAttributes.URL)
		*/
	case "build":
		if event.BuildStatus == "failed" {
			return fmt.Sprintf("🚀 %s job - %s : %s ❌\n%s",
				event.Project.Name, event.BuildName, event.BuildStatus, event.ObjectAttributes.URL)
		} else {
			return ""
		}
		/*
			if event.BuildStatus == "success" {
				return fmt.Sprintf("🚀 %s job - %s : %s ✅\n%s",
					event.Project.Name, event.BuildName, event.BuildStatus, event.ObjectAttributes.URL)
			}
		*/
	default:
		// Unknown type, do not send
		log.Printf("GitLab webhook - unhandled object_kind: %s (project: %s)", event.ObjectKind, event.Project.Name)
		return ""
	}
	return ""
}

// handleGitLabWebhook handles incoming GitLab webhook requests
func handleGitLabWebhook(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received GitLab webhook request")

		// Read the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading GitLab webhook body: %v", err)
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Parse the GitLab webhook payload
		var event GitLabEvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Printf("Error parsing GitLab webhook JSON: %v", err)
			http.Error(w, "Error parsing JSON", http.StatusBadRequest)
			return
		}

		log.Printf("GitLab webhook - ObjectKind: %s, Project: %s",
			event.ObjectKind, event.Project.Name)

		// Generate formatted message from event
		message := generateGitLabMessage(event)
		if message != "" {
			title := config.LarkMessageTitle
			if title == "" {
				title = "GitLab"
			}
			sendNotification(config, message, title)
		}

		// Return empty payload with 200 status code
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}
}
