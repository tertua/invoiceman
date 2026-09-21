package mail

import (
	"errors"
	"testing"
)

func TestNewFromEnvRequiresHost(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	mailer, err := NewFromEnv()
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got mailer=%v err=%v", mailer, err)
	}
}

func TestNewFromEnvDefaultsFromUser(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USER", "sender@example.com")
	t.Setenv("SMTP_PASS", "secret")
	t.Setenv("SMTP_FROM", "")

	mailer, err := NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	smtpMailer, ok := mailer.(*SMTPMailer)
	if !ok {
		t.Fatalf("expected *SMTPMailer, got %T", mailer)
	}
	if smtpMailer.from != "sender@example.com" || smtpMailer.port != "587" {
		t.Fatalf("unexpected mailer config: %+v", smtpMailer)
	}
}
