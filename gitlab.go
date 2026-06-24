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

// isBranchAllowed checks if a branch is in the allowed list
// If allowedBranches is empty, all branches are allowed
func isBranchAllowed(branch string, allowedBranches []string) bool {
	if len(allowedBranches) == 0 {
		return true
	}
	for _, allowed := range allowedBranches {
		if allowed == branch {
			return true
		}
	}
	return false
}

// isStatusAllowed checks if a status is in the allowed list
// If allowedStatuses is empty, only "failed" status is allowed (default behavior)
func isStatusAllowed(status string, allowedStatuses []string, notifyAll bool) bool {
	if notifyAll {
		return true
	}
	if len(allowedStatuses) == 0 {
		// Default: only notify failed
		return status == "failed"
	}
	for _, allowed := range allowedStatuses {
		if allowed == status {
			return true
		}
	}
	return false
}

// generateGitLabMessage generates a formatted message from GitLab event
func generateGitLabMessage(event GitLabEvent, config Config) string {
	// Check if this event type is allowed based on object_kind
	eventType := fmt.Sprintf("gitlab.%s", event.ObjectKind)
	if !isEventAllowed(eventType) {
		return ""
	}

	switch event.ObjectKind {
	case "push":
		if event.TotalCommitCount > 0 {
			// Extract branch name from ref (refs/heads/main -> main)
			branch := event.Ref
			if len(branch) > 11 && branch[:11] == "refs/heads/" {
				branch = branch[11:]
			}

			// Check if branch is allowed
			if !isBranchAllowed(branch, config.GitLabBranches) {
				log.Printf("[GitLab] Push event skipped - branch '%s' not in allowed list (project: %s)", branch, event.Project.Name)
				return ""
			}

			return fmt.Sprintf("🔨 New push by %s to %s (%s)\n%s/-/commits/%s",
				event.UserUsername, event.Project.Name, event.Ref, event.Project.WebURL, event.Ref)
		}
	case "pipeline":
		// Handle pipeline events
		pipelineRef := event.ObjectAttributes.Ref
		pipelineStatus := event.ObjectAttributes.Status

		// Check if branch is allowed
		if !isBranchAllowed(pipelineRef, config.GitLabBranches) {
			log.Printf("[GitLab] Pipeline event skipped - branch '%s' not in allowed list (project: %s, status: %s)",
				pipelineRef, event.Project.Name, pipelineStatus)
			return ""
		}

		// Check if status is allowed
		if !isStatusAllowed(pipelineStatus, config.GitLabStatuses, config.GitLabNotifyAll) {
			log.Printf("[GitLab] Pipeline event skipped - status '%s' not in allowed list (project: %s, branch: %s)",
				pipelineStatus, event.Project.Name, pipelineRef)
			return ""
		}

		// Generate message based on status
		switch pipelineStatus {
		case "failed":
			return fmt.Sprintf("❌ Pipeline FAILED on %s/%s\n%s",
				event.Project.Name, pipelineRef, event.ObjectAttributes.URL)
		case "success":
			return fmt.Sprintf("✅ Pipeline succeeded on %s/%s\n%s",
				event.Project.Name, pipelineRef, event.ObjectAttributes.URL)
		case "running":
			return fmt.Sprintf("🔄 Pipeline running on %s/%s\n%s",
				event.Project.Name, pipelineRef, event.ObjectAttributes.URL)
		case "pending":
			return fmt.Sprintf("⏳ Pipeline pending on %s/%s\n%s",
				event.Project.Name, pipelineRef, event.ObjectAttributes.URL)
		case "canceled":
			return fmt.Sprintf("🚫 Pipeline canceled on %s/%s\n%s",
				event.Project.Name, pipelineRef, event.ObjectAttributes.URL)
		case "skipped":
			return fmt.Sprintf("⏭️ Pipeline skipped on %s/%s\n%s",
				event.Project.Name, pipelineRef, event.ObjectAttributes.URL)
		default:
			return fmt.Sprintf("ℹ️ Pipeline %s on %s/%s\n%s",
				pipelineStatus, event.Project.Name, pipelineRef, event.ObjectAttributes.URL)
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
		// Check if status is allowed
		if !isStatusAllowed(event.BuildStatus, config.GitLabStatuses, config.GitLabNotifyAll) {
			log.Printf("[GitLab] Build event skipped - status '%s' not in allowed list (project: %s, job: %s)",
				event.BuildStatus, event.Project.Name, event.BuildName)
			return ""
		}

		// Generate message based on status
		switch event.BuildStatus {
		case "failed":
			failureInfo := ""
			if event.BuildFailureReason != "" {
				failureInfo = fmt.Sprintf(" (%s)", event.BuildFailureReason)
			}
			return fmt.Sprintf("❌ Job FAILED: %s - %s%s\n%s",
				event.Project.Name, event.BuildName, failureInfo, event.ObjectAttributes.URL)
		case "success":
			return fmt.Sprintf("✅ Job succeeded: %s - %s\n%s",
				event.Project.Name, event.BuildName, event.ObjectAttributes.URL)
		case "running":
			return fmt.Sprintf("🔄 Job running: %s - %s\n%s",
				event.Project.Name, event.BuildName, event.ObjectAttributes.URL)
		case "pending":
			return fmt.Sprintf("⏳ Job pending: %s - %s\n%s",
				event.Project.Name, event.BuildName, event.ObjectAttributes.URL)
		case "canceled":
			return fmt.Sprintf("🚫 Job canceled: %s - %s\n%s",
				event.Project.Name, event.BuildName, event.ObjectAttributes.URL)
		default:
			return fmt.Sprintf("ℹ️ Job %s: %s - %s\n%s",
				event.BuildStatus, event.Project.Name, event.BuildName, event.ObjectAttributes.URL)
		}
	default:
		// Unknown type, do not send
		log.Printf("[GitLab] Unhandled object_kind: %s (project: %s)", event.ObjectKind, event.Project.Name)
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
		message := generateGitLabMessage(event, config)
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
