package events

import (
	"github.com/nyaruka/goflow/flows"
)

func init() {
	registerType(TypeFeedbackRequested, func() flows.Event { return &FeedbackRequestedEvent{} })
}

// TypeFeedbackRequested is a constant for incoming messages
const TypeFeedbackRequested string = "feedback_requested"

// TypeFeedbackRequested events are created when an action wants to send a reply to the current contact.
//
//   {
//     "type": "feedback_requested",
//     "created_on": "2006-01-02T15:04:05Z",
//     "feedback_request": {
//       "uuid": "2d611e17-fb22-457f-b802-b8f7ec5cda5b",
//       "channel": {"uuid": "61602f3e-f603-4c70-8a8f-c477505bf4bf", "name": "Twilio"},
//       "urn": "tel:+12065551212",
//       "star_rating_question": "How would you rate us?",
//       "comment_question": "Please, leave a comment.",
//     }
//   }
//
// @event feedback_requested
type FeedbackRequestedEvent struct {
	BaseEvent

	FeedbackRequest *flows.FeedbackRequest `json:"feedback_request" validate:"required,dive"`
}

// NewMsgCreated creates a new outgoing msg event to a single contact
func NewFeedbackRequestCreated(feedback *flows.FeedbackRequest) *FeedbackRequestedEvent {
	return &FeedbackRequestedEvent{
		BaseEvent: NewBaseEvent(TypeFeedbackRequested),
		FeedbackRequest: feedback,
	}
}
