package events

import (
	"github.com/nyaruka/goflow/assets"
	"github.com/nyaruka/goflow/flows"
)

func init() {
	registerType(TypeTicketOpened, func() flows.Event { return &TicketOpenedEvent{} })
}

// TypeTicketOpened is the type for our ticket opened events
const TypeTicketOpened string = "ticket_opened"

type Ticket struct {
	UUID     flows.TicketUUID       `json:"uuid"                   validate:"required,uuid4"`
	Topic    *assets.TopicReference `json:"topic"                  validate:"omitempty"`
	Body     string                 `json:"body"`
	Assignee *assets.UserReference  `json:"assignee,omitempty"     validate:"omitempty"`
}

// TicketOpenedEvent events are created when a new ticket is opened.
//
//	{
//	  "type": "ticket_opened",
//	  "created_on": "2006-01-02T15:04:05Z",
//	  "ticket": {
//	    "uuid": "2e677ae6-9b57-423c-b022-7950503eef35",
//	    "topic": {
//	      "uuid": "add17edf-0b6e-4311-bcd7-a64b2a459157",
//	      "name": "Weather"
//	    },
//	    "body": "Where are my cookies?",
//	    "assignee": {"email": "bob@nyaruka.com", "name": "Bob"}
//	  }
//	}
//
// @event ticket_opened
type TicketOpenedEvent struct {
	BaseEvent

	Ticket *Ticket `json:"ticket"`
}

// NewTicketOpened returns a new ticket opened event
func NewTicketOpened(ticket *flows.Ticket) *TicketOpenedEvent {
	return &TicketOpenedEvent{
		BaseEvent: NewBaseEvent(TypeTicketOpened),
		Ticket: &Ticket{
			UUID:     ticket.UUID(),
			Topic:    ticket.Topic().Reference(),
			Body:     ticket.Body(),
			Assignee: ticket.Assignee().Reference(),
		},
	}
}
