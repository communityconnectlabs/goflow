package events

import (
	"github.com/greatnonprofits-nfp/goflow/flows"
)

func init() {
	RegisterType(TypeEmailCreated, func() flows.Event { return &EmailCreatedEvent{} })
}

// TypeEmailCreated is our type for the email event
const TypeEmailCreated string = "email_created"

// EmailCreatedEvent events are created when an action wants to send an email.
//
//   {
//     "type": "email_created",
//     "created_on": "2006-01-02T15:04:05Z",
//     "addresses": ["foo@bar.com"],
//     "subject": "Your activation token",
//     "body": "Your activation token is AAFFKKEE"
//   }
//
// @event email_created
type EmailCreatedEvent struct {
	BaseEvent

	Addresses []string `json:"addresses" validate:"required,min=1"`
	Subject   string   `json:"subject" validate:"required"`
	Body      string   `json:"body"`
}

// NewEmailCreatedEvent returns a new email event with the passed in subject, body and emails
func NewEmailCreatedEvent(addresses []string, subject string, body string) *EmailCreatedEvent {
	return &EmailCreatedEvent{
		BaseEvent: NewBaseEvent(TypeEmailCreated),
		Addresses: addresses,
		Subject:   subject,
		Body:      body,
	}
}
