package smtpx

import (
	"github.com/go-mail/mail"
)

// Send an email using SMTP
func Send(host string, port int, username, password, from string, recipients []string, subject, body string, attachments []string) error {
	return currentSender.Send(host, port, username, password, from, recipients, subject, body, attachments)
}

// Sender is anything that can send an email
type Sender interface {
	Send(host string, port int, username, password, from string, recipients []string, subject, body string, attachments []string) error
}

type defaultSender struct{}

func (s defaultSender) Send(host string, port int, username, password, from string, recipients []string, subject, body string, attachments []string) error {
	// create our dialer for our org
	d := mail.NewDialer(host, port, username, password)

	m := mail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", recipients...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	for _, filepath := range attachments {
		m.Attach(filepath)
	}

	return d.DialAndSend(m)
}

// DefaultSender is the default SMTP sender
var DefaultSender Sender = defaultSender{}
var currentSender = DefaultSender

// SetSender sets the sender used by Send
func SetSender(sender Sender) {
	currentSender = sender
}
