package events

import (
	"github.com/nyaruka/gocommon/urns"
	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/utils"
)

func init() {
	RegisterType(TypeBroadcastCreated, func() flows.Event { return &BroadcastCreatedEvent{} })
}

// TypeBroadcastCreated is a constant for outgoing message events
const TypeBroadcastCreated string = "broadcast_created"

// BroadcastTranslation is the broadcast content in a particular language
type BroadcastTranslation struct {
	Text         string             `json:"text"`
	Attachments  []utils.Attachment `json:"attachments,omitempty"`
	QuickReplies []string           `json:"quick_replies,omitempty"`
}

// BroadcastCreatedEvent events are created when an action wants to send a message to other contacts.
//
//   {
//     "type": "broadcast_created",
//     "created_on": "2006-01-02T15:04:05Z",
//     "translations": {
//       "eng": {
//         "text": "hi, what's up",
//         "attachments": [],
//         "quick_replies": ["All good", "Got 99 problems"]
//       },
//       "spa": {
//         "text": "Que pasa",
//         "attachments": [],
//         "quick_replies": ["Todo bien", "Tengo 99 problemas"]
//       }
//     },
//     "base_language": "eng",
//     "urns": ["tel:+12065551212"],
//     "contacts": [{"uuid": "0e06f977-cbb7-475f-9d0b-a0c4aaec7f6a", "name": "Bob"}]
//   }
//
// @event broadcast_created
type BroadcastCreatedEvent struct {
	BaseEvent

	Translations map[utils.Language]*BroadcastTranslation `json:"translations,min=1" validate:"dive"`
	BaseLanguage utils.Language                           `json:"base_language" validate:"required"`
	URNs         []urns.URN                               `json:"urns,omitempty" validate:"dive,urn"`
	Contacts     []*flows.ContactReference                `json:"contacts,omitempty" validate:"dive"`
	Groups       []*assets.GroupReference                 `json:"groups,omitempty" validate:"dive"`
}

// NewBroadcastCreatedEvent creates a new outgoing msg event for the given recipients
func NewBroadcastCreatedEvent(translations map[utils.Language]*BroadcastTranslation, baseLanguage utils.Language, urns []urns.URN, contacts []*flows.ContactReference, groups []*assets.GroupReference) *BroadcastCreatedEvent {
	event := BroadcastCreatedEvent{
		BaseEvent:    NewBaseEvent(TypeBroadcastCreated),
		Translations: translations,
		BaseLanguage: baseLanguage,
		URNs:         urns,
		Contacts:     contacts,
		Groups:       groups,
	}
	return &event
}

var _ flows.Event = (*BroadcastCreatedEvent)(nil)
