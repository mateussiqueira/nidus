package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailConfig struct {
	ResendAPIKey string
	FromEmail    string
	FromName     string
	Enabled      bool
}

type ResendPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

func loadEmailConfig() EmailConfig {
	cfg := EmailConfig{
		ResendAPIKey: os.Getenv("RESEND_API_KEY"),
		FromEmail:    os.Getenv("EMAIL_FROM"),
		FromName:     os.Getenv("EMAIL_FROM_NAME"),
	}
	if cfg.ResendAPIKey != "" && cfg.FromEmail != "" {
		cfg.Enabled = true
		if cfg.FromName == "" {
			cfg.FromName = "StackRun"
		}
	}
	return cfg
}

func (cfg EmailConfig) sendDeployNotification(toEmail, projectName, status, deployURL, branch string, duration time.Duration) {
	if !cfg.Enabled || toEmail == "" {
		return
	}

	from := fmt.Sprintf("%s <%s>", cfg.FromName, cfg.FromEmail)
	isSuccess := status == "success"
	emoji := "✅"
	statusLabel := "Sucesso"
	color := "#22c55e"

	if !isSuccess {
		emoji = "❌"
		statusLabel = "Falha"
		color = "#ef4444"
	}

	subject := fmt.Sprintf("%s Deploy %s — %s", emoji, statusLabel, projectName)
	durationStr := fmt.Sprintf("%.0fs", duration.Seconds())

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family:-apple-system,BlinkMacSystemFont,sans-serif;background:#09090b;padding:40px 20px">
<table width="100%%" cellpadding="0" cellspacing="0"><tr><td align="center">
<table style="max-width:480px;width:100%%;background:#18181b;border-radius:12px;overflow:hidden;border:1px solid #27272a">
<tr><td style="padding:32px 24px;text-align:center;background:%s">
<span style="font-size:48px">%s</span>
<h1 style="color:#fff;font-size:20px;margin:12px 0 0;font-weight:600">Deploy %s</h1>
</td></tr>
<tr><td style="padding:24px">
<table width="100%%" cellpadding="0" cellspacing="0" style="font-size:14px;color:#a1a1aa">
<tr><td style="padding:8px 0;color:#fafafa;font-weight:500">%s</td></tr>
<tr><td style="padding:4px 0"><strong style="color:#fafafa">Status:</strong> <span style="color:%s">%s</span></td></tr>
<tr><td style="padding:4px 0"><strong style="color:#fafafa">Branch:</strong> %s</td></tr>
<tr><td style="padding:4px 0"><strong style="color:#fafafa">Duração:</strong> %s</td></tr>
%s
</table>
</td></tr>
<tr><td style="padding:16px 24px;border-top:1px solid #27272a;text-align:center;font-size:12px;color:#52525b">
StackRun — Sua PaaS pessoal
</td></tr>
</table>
</td></tr></table>
</body>
</html>`, color, emoji, statusLabel, projectName, color, statusLabel, branch, durationStr,
		mapIf(deployURL != "", fmt.Sprintf(`<tr><td style="padding:4px 0"><strong style="color:#fafafa">URL:</strong> <a href="%s" style="color:#22c55e">%s</a></td></tr>`, deployURL, deployURL), ""))

	payload := ResendPayload{
		From:    from,
		To:      toEmail,
		Subject: subject,
		HTML:    html,
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post("https://api.resend.com/emails",
		"application/json",
		bytes.NewReader(body))
	if err != nil {
		log.Printf("[email] Failed to send: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("[email] API error: %d", resp.StatusCode)
	} else {
		log.Printf("[email] Sent %s notification for %s", status, projectName)
	}
}

func mapIf(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

// fetchUserEmail retrieves the user's email from the database for a project
func fetchUserEmail(projectID string, pool *pgxpool.Pool) string {
	if pool == nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var email string
	err := pool.QueryRow(ctx,
		`SELECT u.email FROM users u
		 JOIN projects p ON p.user_id = u.id
		 WHERE p.id = $1`, projectID).Scan(&email)
	if err != nil {
		return ""
	}
	return email
}
