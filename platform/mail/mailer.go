package mail

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"mime/multipart"
	"net"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/tertua/tupay/pkg/configs"
)

var ErrNotConfigured = errors.New("mail provider is not configured")

type Mailer interface {
	Send(to, subject, textBody string) error
	SendHTML(to, subject, textBody, htmlBody string) error
}

type SMTPMailer struct {
	host   string
	port   string
	from   string
	user   string
	pass   string
	secure bool
}

func NewFromEnv() (Mailer, error) {
	cfg := configs.Get().Mail
	host := cfg.SMTPHost
	if host == "" {
		return nil, ErrNotConfigured
	}

	port := cfg.SMTPPort
	if port == "" {
		port = "587"
	}
	if _, err := strconv.Atoi(port); err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT: %w", err)
	}

	from := cfg.SMTPFrom
	if from == "" {
		from = cfg.SMTPUser
	}
	if from == "" {
		return nil, errors.New("SMTP_FROM or SMTP_USER is required")
	}

	return &SMTPMailer{
		host:   host,
		port:   port,
		from:   from,
		user:   cfg.SMTPUser,
		pass:   cfg.SMTPPass,
		secure: port == "465",
	}, nil
}

func (m *SMTPMailer) Send(to, subject, textBody string) error {
	return m.deliver(to, subject, textBody, "")
}

// SendHTML delivers a multipart/alternative message: plain text for
// deliverability plus an HTML part for capable clients. An empty htmlBody
// falls back to text-only.
func (m *SMTPMailer) SendHTML(to, subject, textBody, htmlBody string) error {
	return m.deliver(to, subject, textBody, htmlBody)
}

func (m *SMTPMailer) deliver(to, subject, textBody, htmlBody string) error {
	if strings.TrimSpace(to) == "" {
		return errors.New("recipient is required")
	}
	message, err := buildMessage(m.from, to, subject, textBody, htmlBody)
	if err != nil {
		return err
	}
	address := net.JoinHostPort(m.host, m.port)

	if m.secure {
		return m.sendTLS(address, to, message)
	}

	auth := smtp.Auth(nil)
	if m.user != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}
	return smtp.SendMail(address, auth, m.from, []string{to}, message)
}

// buildMessage assembles headers plus a text-only or multipart body.
func buildMessage(from, to, subject, textBody, htmlBody string) ([]byte, error) {
	headers := []string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
	}
	if strings.TrimSpace(htmlBody) == "" {
		headers = append(headers, "Content-Type: text/plain; charset=UTF-8", "", textBody)
		return []byte(strings.Join(headers, "\r\n")), nil
	}
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	headers = append(headers, "Content-Type: multipart/alternative; boundary="+writer.Boundary(), "")
	buf.WriteString(strings.Join(headers, "\r\n"))
	textPart, err := writer.CreatePart(map[string][]string{
		"Content-Type": {"text/plain; charset=UTF-8"},
	})
	if err != nil {
		return nil, err
	}
	if _, err := textPart.Write([]byte(textBody)); err != nil {
		return nil, err
	}
	htmlPart, err := writer.CreatePart(map[string][]string{
		"Content-Type": {"text/html; charset=UTF-8"},
	})
	if err != nil {
		return nil, err
	}
	if _, err := htmlPart.Write([]byte(htmlBody)); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *SMTPMailer) sendTLS(address, to string, message []byte) error {
	conn, err := tls.Dial("tcp", address, &tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12})
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer func() { _ = client.Close() }()
	if m.user != "" {
		if err := client.Auth(smtp.PlainAuth("", m.user, m.pass, m.host)); err != nil {
			return err
		}
	}
	if err := client.Mail(m.from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
