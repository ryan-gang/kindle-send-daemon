package mail

import "github.com/ryan-gang/kindle-send-daemon/internal/config"

// MailSender defines the interface for sending emails
type MailSender interface {
	Send(files []string, timeout int) error
}

// SMTPMailSender implements MailSender using SMTP
type SMTPMailSender struct {
	cfg config.ConfigProvider
}

// NewSMTPMailSender creates a new SMTP mail sender
func NewSMTPMailSender(cfg config.ConfigProvider) MailSender {
	return &SMTPMailSender{cfg: cfg}
}
