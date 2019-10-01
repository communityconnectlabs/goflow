package events

import (
	"github.com/greatnonprofits-nfp/goflow/flows"
)

func init() {
	RegisterType(TypeIVRCreated, func() flows.Event { return &IVRCreatedEvent{} })
}

// TypeIVRCreated is a constant for IVR created events
const TypeIVRCreated string = "ivr_created"

// IVRCreatedEvent events are created when an action wants to send an IVR response to the current contact.
//
//   {
//     "type": "ivr_created",
//     "created_on": "2006-01-02T15:04:05Z",
//     "msg": {
//       "uuid": "2d611e17-fb22-457f-b802-b8f7ec5cda5b",
//       "channel": {"uuid": "61602f3e-f603-4c70-8a8f-c477505bf4bf", "name": "Twilio"},
//       "urn": "tel:+12065551212",
//       "text": "hi there",
//       "attachments": ["audio:https://s3.amazon.com/mybucket/attachment.m4a"]
//     }
//   }
//
// @event ivr_created
type IVRCreatedEvent struct {
	BaseEvent

	Msg *flows.MsgOut `json:"msg" validate:"required,dive"`
}

// NewIVRCreatedEvent creates a new IVR created event
func NewIVRCreatedEvent(msg *flows.MsgOut) *IVRCreatedEvent {
	return &IVRCreatedEvent{
		BaseEvent: NewBaseEvent(TypeIVRCreated),
		Msg:       msg,
	}
}
