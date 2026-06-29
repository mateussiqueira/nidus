package mail

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

var db *sql.DB

func Init(database *sql.DB) {
	db = database
}

type Config struct {
	Provider string // "smtp", "sendgrid", "sendmail"
	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	FromName string
	FromEmail string
}

var cfg Config

func Configure(c Config) {
	cfg = c
	if cfg.Provider == "" {
		cfg.Provider = "sendmail"
	}
	if cfg.FromName == "" {
		cfg.FromName = "Nidus"
	}
	if cfg.FromEmail == "" {
		cfg.FromEmail = "noreply@nidus.app"
	}
}

type SendInput struct {
	ToEmail  string            `json:"to"`
	ToName   string            `json:"to_name"`
	TemplateID string          `json:"template_id"`
	Subject  string            `json:"subject"`
	HTMLBody string            `json:"html"`
	TextBody string            `json:"text"`
	Vars     map[string]string `json:"vars"`
}

type SendResult struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Provider string `json:"provider"`
}

type TemplateData struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Subject   string `json:"subject"`
	HTMLBody  string `json:"html"`
	TextBody  string `json:"text"`
	CreatedAt string `json:"created_at"`
}

func GetTemplates() ([]TemplateData, error) {
	rows, err := db.Query("SELECT id, name, subject, html_body, text_body, created_at FROM email_templates ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []TemplateData
	for rows.Next() {
		var t TemplateData
		var createdAt time.Time
		if err := rows.Scan(&t.ID, &t.Name, &t.Subject, &t.HTMLBody, &t.TextBody, &createdAt); err != nil {
			continue
		}
		t.CreatedAt = createdAt.Format(time.RFC3339)
		templates = append(templates, t)
	}
	return templates, nil
}

func renderTemplate(tmpl string, vars map[string]string) (string, error) {
	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return tmpl, err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return tmpl, err
	}
	return buf.String(), nil
}

func Send(input SendInput) (*SendResult, error) {
	id := uuid.New().String(); _ = id
	status := "pending"
	subject := input.Subject
	htmlBody := input.HTMLBody
	textBody := input.TextBody

	// Load template if specified
	if input.TemplateID != "" {
		var tSubject, tHTML, tText string
		err := db.QueryRow("SELECT subject, html_body, text_body FROM email_templates WHERE id = $1", input.TemplateID).
			Scan(&tSubject, &tHTML, &tText)
		if err == nil {
			if subject == "" {
				subject = tSubject
			}
			if htmlBody == "" {
				htmlBody = tHTML
			}
			if textBody == "" {
				textBody = tText
			}
		}
	}

	// Render template variables
	if input.Vars != nil && len(input.Vars) > 0 {
		var err error
		subject, err = renderTemplate(subject, input.Vars)
		if err != nil {
			subject = input.Subject
		}
		htmlBody, err = renderTemplate(htmlBody, input.Vars)
		textBody, _ = renderTemplate(textBody, input.Vars)
	}

	// Send via configured provider
	var err error
	switch cfg.Provider {
	case "sendmail":
		err = sendViaSendmail(input.ToEmail, subject, htmlBody, textBody)
	case "smtp":
		err = sendViaSMTP(input.ToEmail, subject, htmlBody)
	default:
		err = fmt.Errorf("unknown provider: %s", cfg.Provider)
	}

	// Log result
	logID := uuid.New().String()
	now := time.Now()
	if err != nil {
		status = "failed"
		db.Exec(`INSERT INTO email_logs (id, template_id, to_email, to_name, subject, html_body, text_body, status, provider, error_message, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			logID, input.TemplateID, input.ToEmail, input.ToName, subject, htmlBody, textBody, status, cfg.Provider, err.Error(), now)
		return &SendResult{ID: logID, Status: status, Provider: cfg.Provider}, err
	}

	status = "sent"
	db.Exec(`INSERT INTO email_logs (id, template_id, to_email, to_name, subject, html_body, text_body, status, provider, sent_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		logID, input.TemplateID, input.ToEmail, input.ToName, subject, htmlBody, textBody, status, cfg.Provider, now, now)

	return &SendResult{ID: logID, Status: status, Provider: cfg.Provider}, nil
}

func sendViaSendmail(to, subject, htmlBody, _ string) error {
	from := fmt.Sprintf("%s <%s>", cfg.FromName, cfg.FromEmail)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s", from, to, subject, htmlBody)
	cmd := exec.Command("/usr/sbin/sendmail", "-t")
	cmd.Stdin = strings.NewReader(msg)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func sendViaSMTP(to, subject, htmlBody string) error {
	from := cfg.FromEmail
	msg := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		cfg.FromName, from, to, subject, htmlBody)

	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	addr := fmt.Sprintf("%s:%s", cfg.SMTPHost, cfg.SMTPPort)
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}
