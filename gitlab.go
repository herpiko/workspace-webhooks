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
	switch event.ObjectKind {
	/*
		case "push":
			if event.TotalCommitCount > 0 {
				// branchName := strings.TrimPrefix(event.Ref, "refs/heads/")
				// branchLink := fmt.Sprintf("%s/-/tree/%s", event.Project.WebURL, url.QueryEscape(branchName))
				return fmt.Sprintf("🔨 New push by %s to %s: %s",
					event.UserUsername, event.Project.Name, event.Ref)
			}
	*/
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
