package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// GitHubRepository represents repository information in GitHub webhooks
type GitHubRepository struct {
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

// GitHubUser represents user information in GitHub webhooks
type GitHubUser struct {
	Login string `json:"login"`
}

// GitHubPullRequest represents a pull request in GitHub webhooks
type GitHubPullRequest struct {
	Number  int        `json:"number"`
	Title   string     `json:"title"`
	HTMLURL string     `json:"html_url"`
	State   string     `json:"state"`
	User    GitHubUser `json:"user"`
}

// GitHubPushEvent represents a GitHub push event
type GitHubPushEvent struct {
	Ref        string           `json:"ref"`
	Repository GitHubRepository `json:"repository"`
	Pusher     struct {
		Name string `json:"name"`
	} `json:"pusher"`
	Commits []struct {
		Message string `json:"message"`
	} `json:"commits"`
}

// GitHubPullRequestEvent represents a GitHub pull request event
type GitHubPullRequestEvent struct {
	Action      string            `json:"action"`
	PullRequest GitHubPullRequest `json:"pull_request"`
	Repository  GitHubRepository  `json:"repository"`
}

// GitHubIssueCommentEvent represents a GitHub issue comment event
type GitHubIssueCommentEvent struct {
	Action     string            `json:"action"`
	Issue      GitHubPullRequest `json:"issue"`
	Comment    struct {
		HTMLURL string     `json:"html_url"`
		User    GitHubUser `json:"user"`
	} `json:"comment"`
	Repository GitHubRepository `json:"repository"`
}

// GitHubWorkflowRunEvent represents a GitHub workflow run event
type GitHubWorkflowRunEvent struct {
	Action      string `json:"action"`
	WorkflowRun struct {
		Name       string `json:"name"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		HTMLURL    string `json:"html_url"`
	} `json:"workflow_run"`
	Repository GitHubRepository `json:"repository"`
}

// generateGitHubPushMessage generates a formatted message from GitHub push event
func generateGitHubPushMessage(event GitHubPushEvent) string {
	if len(event.Commits) > 0 {
		return fmt.Sprintf("🔨 New push by %s to %s\nBranch: %s\n%s",
			event.Pusher.Name, event.Repository.Name, event.Ref, event.Repository.HTMLURL)
	}
	return ""
}

// generateGitHubPullRequestMessage generates a formatted message from GitHub PR event
func generateGitHubPullRequestMessage(event GitHubPullRequestEvent) string {
	switch event.Action {
	case "opened", "reopened":
		return fmt.Sprintf("🔥 New PR opened by %s: %s - %s",
			event.PullRequest.User.Login, event.PullRequest.Title, event.PullRequest.HTMLURL)
	/*
	case "closed":
		if event.PullRequest.Merged {
			return fmt.Sprintf("🎉 Pull request MERGED by %s: %s\n%s",
				event.PullRequest.User.Login, event.PullRequest.Title, event.PullRequest.HTMLURL)
		}
		return fmt.Sprintf("❌ Pull request closed by %s: %s\n%s",
			event.PullRequest.User.Login, event.PullRequest.Title, event.PullRequest.HTMLURL)
	*/
	default:
		return ""
	}
}

// generateGitHubIssueCommentMessage generates a formatted message from GitHub issue comment event
func generateGitHubIssueCommentMessage(event GitHubIssueCommentEvent) string {
	/*
	if event.Action == "created" {
		return fmt.Sprintf("💬 New comment by %s\n%s",
			event.Comment.User.Login, event.Comment.HTMLURL)
	}
	*/
	return ""
}

// generateGitHubWorkflowRunMessage generates a formatted message from GitHub workflow run event
func generateGitHubWorkflowRunMessage(event GitHubWorkflowRunEvent) string {
	if event.Action == "completed" {
		if event.WorkflowRun.Conclusion == "failure" {
			return fmt.Sprintf("🚀 %s job - %s : %s ❌\n%s",
				event.Repository.Name, event.WorkflowRun.Name, event.WorkflowRun.Conclusion, event.WorkflowRun.HTMLURL)
		}
		/*
		if event.WorkflowRun.Conclusion == "success" {
			return fmt.Sprintf("🚀 %s job - %s : %s ✅\n%s",
				event.Repository.Name, event.WorkflowRun.Name, event.WorkflowRun.Conclusion, event.WorkflowRun.HTMLURL)
		}
		*/
	}
	return ""
}

// handleGitHubWebhook handles incoming GitHub webhook requests
func handleGitHubWebhook(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received GitHub webhook request")

		// Get the GitHub event type from headers
		eventType := r.Header.Get("X-GitHub-Event")
		log.Printf("GitHub Event Type: %s", eventType)

		// Read the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading GitHub webhook body: %v", err)
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var message string

		// Handle different event types
		switch eventType {
		case "push":
			var event GitHubPushEvent
			if err := json.Unmarshal(body, &event); err != nil {
				log.Printf("Error parsing GitHub push event JSON: %v", err)
				http.Error(w, "Error parsing JSON", http.StatusBadRequest)
				return
			}
			message = generateGitHubPushMessage(event)

		case "pull_request":
			var event GitHubPullRequestEvent
			if err := json.Unmarshal(body, &event); err != nil {
				log.Printf("Error parsing GitHub pull request event JSON: %v", err)
				http.Error(w, "Error parsing JSON", http.StatusBadRequest)
				return
			}
			message = generateGitHubPullRequestMessage(event)

		case "issue_comment":
			var event GitHubIssueCommentEvent
			if err := json.Unmarshal(body, &event); err != nil {
				log.Printf("Error parsing GitHub issue comment event JSON: %v", err)
				http.Error(w, "Error parsing JSON", http.StatusBadRequest)
				return
			}
			message = generateGitHubIssueCommentMessage(event)

		case "workflow_run":
			var event GitHubWorkflowRunEvent
			if err := json.Unmarshal(body, &event); err != nil {
				log.Printf("Error parsing GitHub workflow run event JSON: %v", err)
				http.Error(w, "Error parsing JSON", http.StatusBadRequest)
				return
			}
			message = generateGitHubWorkflowRunMessage(event)

		default:
			log.Printf("Unhandled GitHub event type: %s", eventType)
		}

		// Send message to notification channels if we have a message to send
		if message != "" {
			title := config.LarkMessageTitle
			if title == "" {
				title = "GitHub"
			}
			sendNotification(config, message, title)
		}

		// Return empty payload with 200 status code
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}
}
