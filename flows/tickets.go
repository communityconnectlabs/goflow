package flows

import (
	"github.com/nyaruka/gocommon/jsonx"
	"github.com/nyaruka/gocommon/uuids"
	"github.com/nyaruka/goflow/assets"
	"github.com/nyaruka/goflow/envs"
	"github.com/nyaruka/goflow/excellent/types"
	"github.com/nyaruka/goflow/utils"
)

// TicketUUID is the UUID of a ticket
type TicketUUID uuids.UUID

// Ticket is a ticket in a ticketing system
type Ticket struct {
	uuid     TicketUUID
	topic    *Topic
	body     string
	assignee *User
}

// NewTicket creates a new ticket
func NewTicket(uuid TicketUUID, topic *Topic, body string, assignee *User) *Ticket {
	return &Ticket{
		uuid:     uuid,
		topic:    topic,
		body:     body,
		assignee: assignee,
	}
}

// OpenTicket creates a new ticket. Used by ticketing services to open a new ticket.
func OpenTicket(topic *Topic, body string, assignee *User) *Ticket {
	return NewTicket(TicketUUID(uuids.New()), topic, body, assignee)
}

func (t *Ticket) UUID() TicketUUID { return t.uuid }
func (t *Ticket) Topic() *Topic    { return t.topic }
func (t *Ticket) Body() string     { return t.body }
func (t *Ticket) Assignee() *User  { return t.assignee }

// Context returns the properties available in expressions
//
//	uuid:text -> the UUID of the ticket
//	subject:text -> the subject of the ticket
//	body:text -> the body of the ticket
//
// @context ticket
func (t *Ticket) Context(env envs.Environment) map[string]types.XValue {
	return map[string]types.XValue{
		"uuid":     types.NewXText(string(t.uuid)),
		"topic":    Context(env, t.topic),
		"body":     types.NewXText(t.body),
		"assignee": Context(env, t.assignee),
	}
}

//------------------------------------------------------------------------------------------
// JSON Encoding / Decoding
//------------------------------------------------------------------------------------------

type ticketEnvelope struct {
	UUID     TicketUUID             `json:"uuid"                   validate:"required,uuid4"`
	Topic    *assets.TopicReference `json:"topic"                  validate:"omitempty,dive"`
	Body     string                 `json:"body"`
	Assignee *assets.UserReference  `json:"assignee,omitempty"     validate:"omitempty,dive"`
}

// ReadTicket decodes a contact from the passed in JSON. If the topic or assigned user can't
// be found in the assets, we report the missing asset and return ticket without those.
func ReadTicket(sa SessionAssets, data []byte, missing assets.MissingCallback) (*Ticket, error) {
	e := &ticketEnvelope{}

	if err := utils.UnmarshalAndValidate(data, e); err != nil {
		return nil, err
	}

	var topic *Topic
	if e.Topic != nil {
		topic = sa.Topics().Get(e.Topic.UUID)
		if topic == nil {
			missing(e.Topic, nil)
		}
	}

	var assignee *User
	if e.Assignee != nil {
		assignee = sa.Users().Get(e.Assignee.Email)
		if assignee == nil {
			missing(e.Assignee, nil)
		}
	}

	return &Ticket{
		uuid:     e.UUID,
		topic:    topic,
		body:     e.Body,
		assignee: assignee,
	}, nil
}

// MarshalJSON marshals this ticket into JSON
func (t *Ticket) MarshalJSON() ([]byte, error) {
	var topicRef *assets.TopicReference
	if t.topic != nil {
		topicRef = t.topic.Reference()
	}

	var assigneeRef *assets.UserReference
	if t.assignee != nil {
		assigneeRef = t.assignee.Reference()
	}

	return jsonx.Marshal(&ticketEnvelope{
		UUID:     t.uuid,
		Topic:    topicRef,
		Body:     t.body,
		Assignee: assigneeRef,
	})
}
