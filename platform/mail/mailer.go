package mail

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

var ErrNotConfigured = errors.New("mail provider is not configured")

type Mailer interface {
	Send(to, subject, textBody string) error
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
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if host == "" {
		return nil, ErrNotConfigured
	}

	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "587"
	}
	if _, err := strconv.Atoi(port); err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT: %w", err)
	}

	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		from = strings.TrimSpace(os.Getenv("SMTP_USER"))
	}
	if from == "" {
		return nil, errors.New("SMTP_FROM or SMTP_USER is required")
	}

	return &SMTPMailer{
		host:   host,
		port:   port,
		from:   from,
		user:   strings.TrimSpace(os.Getenv("SMTP_USER")),
		pass:   os.Getenv("SMTP_PASS"),
		secure: port == "465",
	}, nil
}

func (m *SMTPMailer) Send(to, subject, textBody string) error {
	if strings.TrimSpace(to) == "" {
		return errors.New("recipient is required")
	}
	message := strings.Join([]string{
		"From: " + m.from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		textBody,
	}, "\r\n")
	address := net.JoinHostPort(m.host, m.port)

	if m.secure {
		return m.sendTLS(address, to, []byte(message))
	}

	auth := smtp.Auth(nil)
	if m.user != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}
	return smtp.SendMail(address, auth, m.from, []string{to}, []byte(message))
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
	defer client.Close()
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
