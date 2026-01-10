# Workspace Webhooks

A flexible webhook middleware that forwards webhooks from various sources (GitHub, GitLab, Grafana) to Lark (Feishu) or Telegram.

## Supported Sources

- **GitHub** - Pull requests, workflow runs, push events, and more
- **GitLab** - Merge requests, pipeline failures, and builds
- **Grafana** - Alert notifications
- **Generic** - Custom webhook payloads

## Configuration

Create a `config.yaml` file with your webhook configurations:

```yaml
configs:
  - source_type: github
    endpoint: /github-webhook-abc123
    lark_webhook_url: https://open.larksuite.com/open-apis/bot/v2/hook/your-webhook-id
    lark_message_title: GitHub

  - source_type: gitlab
    endpoint: /gitlab-webhook-def456
    lark_webhook_url: https://open.larksuite.com/open-apis/bot/v2/hook/your-webhook-id
    lark_message_title: GitLab

  - source_type: grafana
    endpoint: /grafana-webhook-ghi789
    lark_webhook_url: https://open.larksuite.com/open-apis/bot/v2/hook/your-webhook-id
    lark_message_title: Grafana
```

## Running the Server

```bash
./workspace-webhooks -c config.yaml -p 8080
```

## Example Webhooks

### GitHub

**Pull Request:**
```bash
curl -X POST http://localhost:8080/webhook/github-webhook-abc123 \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: pull_request" \
  -d '{
    "action": "opened",
    "pull_request": {
      "number": 123,
      "title": "Add new feature",
      "html_url": "https://github.com/owner/repo/pull/123",
      "user": {"login": "developer123"}
    },
    "repository": {
      "name": "test-repo",
      "html_url": "https://github.com/owner/test-repo"
    }
  }'
```

**Workflow Run (Failed Job):**
```bash
curl -X POST http://localhost:8080/webhook/github-webhook-abc123 \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: workflow_run" \
  -d '{
    "action": "completed",
    "workflow_run": {
      "name": "CI Build",
      "conclusion": "failure",
      "html_url": "https://github.com/owner/repo/actions/runs/12345"
    },
    "repository": {
      "name": "test-repo"
    }
  }'
```

### GitLab

**Merge Request:**
```bash
curl -X POST http://localhost:8080/webhook/gitlab-webhook-def456 \
  -H "Content-Type: application/json" \
  -d '{
    "object_kind": "merge_request",
    "user": {"username": "developer123"},
    "object_attributes": {
      "action": "open",
      "title": "Add new feature",
      "url": "https://gitlab.com/owner/repo/-/merge_requests/123"
    },
    "project": {
      "name": "test-repo",
      "web_url": "https://gitlab.com/owner/test-repo"
    }
  }'
```

### Grafana

```bash
curl -X POST http://localhost:8080/webhook/grafana-webhook-ghi789 -H "Content-Type: application/json" -d '{
    "receiver": "My Super Webhook",
    "status": "firing",
    "orgId": 1,
    "alerts": [
        {
            "status": "firing",
            "labels": {
                "alertname": "High memory usage",
                "team": "blue",
                "zone": "us-1"
            },
            "annotations": {
                "description": "The system has high memory usage",
                "runbook_url": "https://myrunbook.com/runbook/1234",
                "summary": "This alert was triggered for zone us-1"
            },
            "startsAt": "2021-10-12T09:51:03.157076+02:00",
            "endsAt": "0001-01-01T00:00:00Z",
            "generatorURL": "https://play.grafana.org/alerting/1afz29v7z/edit",
            "fingerprint": "c6eadffa33fcdf37",
            "silenceURL": "https://play.grafana.org/alerting/silence/new?alertmanager=grafana&matchers=alertname%3DT2%2Cteam%3Dblue%2Czone%3Dus-1",
            "dashboardURL": "",
            "panelURL": "",
            "values": {
                "B": 44.23943737541908,
                "C": 1
            }
        },
        {
            "status": "firing",
            "labels": {
                "alertname": "High CPU usage",
                "team": "blue",
                "zone": "eu-1"
            },
            "annotations": {
                "description": "The system has high CPU usage",
                "runbook_url": "https://myrunbook.com/runbook/1234",
                "summary": "This alert was triggered for zone eu-1"
            },
            "startsAt": "2021-10-12T09:56:03.157076+02:00",
            "endsAt": "0001-01-01T00:00:00Z",
            "generatorURL": "https://play.grafana.org/alerting/d1rdpdv7k/edit",
            "fingerprint": "bc97ff14869b13e3",
            "silenceURL": "https://play.grafana.org/alerting/silence/new?alertmanager=grafana&matchers=alertname%3DT1%2Cteam%3Dblue%2Czone%3Deu-1",
            "dashboardURL": "",
            "panelURL": "",
            "values": {
                "B": 44.23943737541908,
                "C": 1
            }
        }
    ],
    "groupLabels": {},
    "commonLabels": {
        "team": "blue"
    },
    "commonAnnotations": {},
    "externalURL": "https://play.grafana.org/",
    "version": "1",
    "groupKey": "{}:{}",
    "truncatedAlerts": 0,
    "title": "[FIRING:2]  (blue)",
    "state": "alerting",
    "message": "**Firing**\n\nLabels:\n - alertname = T2\n - team = blue\n - zone = us-1\nAnnotations:\n - description = This is the alert rule checking the second system\n - runbook_url = https://myrunbook.com\n - summary = This is my summary\nSource: https://play.grafana.org/alerting/1afz29v7z/edit\nSilence: https://play.grafana.org/alerting/silence/new?alertmanager=grafana&matchers=alertname%3DT2%2Cteam%3Dblue%2Czone%3Dus-1\n\nLabels:\n - alertname = T1\n - team = blue\n - zone = eu-1\nAnnotations:\nSource: https://play.grafana.org/alerting/d1rdpdv7k/edit\nSilence: https://play.grafana.org/alerting/silence/new?alertmanager=grafana&matchers=alertname%3DT1%2Cteam%3Dblue%2Czone%3Deu-1\n"
}'
```
