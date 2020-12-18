package smtp

import (
	"strings"

	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/utils/smtpx"
)

type service struct {
	smtpClient *smtpx.Client
}

// NewService creates a new SMTP email service
func NewService(smtpURL string) (flows.EmailService, error) {
	c, err := smtpx.NewClientFromURL(smtpURL)
	if err != nil {
		return nil, err
	}

	return &service{smtpClient: c}, nil
}

func (s *service) Send(session flows.Session, addresses []string, subject, body string) error {
	// sending blank emails is a good way to get flagged as a spammer so use placeholder if body is empty
	if strings.TrimSpace(body) == "" {
		body = "(empty body)"
	}

	m := smtpx.NewMessage(addresses, subject, body, "")
	return smtpx.Send(s.smtpClient, m)
}
