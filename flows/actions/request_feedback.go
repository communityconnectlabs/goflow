package actions

import (
	"github.com/nyaruka/goflow/assets"
	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/flows/events"
)

func init() {
	registerType(TypeRequestFeedback, func() flows.Action { return &RequestFeedbackAction{} })
}

// TypeRequestFeedback is the type for the request feedback action
const TypeRequestFeedback string = "request_feedback"

// RequestFeedbackAction can be used to send a request feedback form to contact.
//
// An [event:feedback_requested] event will be created if the form could be sent.
//
//   {
//     "uuid": "8eebd020-1af5-431c-b943-aa670fc74da9",
//     "type": "request_feedback",
//     "star_rating_question": "How would you rate us?",
//     "comment_question": "Please, leave a comment.",
//     "sms_question": "Please rate us from 1 to 5",
//   }
//
// @action request_feedback
type RequestFeedbackAction struct {
	baseAction
	onlineAction

	StarRatingQuestion string `json:"star_rating_question" validate:"required" engine:"localized,evaluated"`
	CommentQuestion    string `json:"comment_question" validate:"required" engine:"localized,evaluated"`
	SMSQuestion        string `json:"sms_question" validate:"required" engine:"localized,evaluated"`
}

// Execute runs this action
func (a *RequestFeedbackAction) Execute(run flows.Run, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	if run.Contact() == nil {
		logEvent(events.NewErrorf("can't execute action in session without a contact"))
		return nil
	}

	// resolve first available destination
	smsPreferredChannelUUID := run.Environment().Config()["sms_default_channel"]
	if smsPreferredChannelUUID == nil {
		smsPreferredChannelUUID = ""
	}
	destinations := run.Contact().ResolveDestinations(false, smsPreferredChannelUUID.(string))

	// create a new feedback request for each URN+channel destination
	for _, dest := range destinations {
		var channelRef *assets.ChannelReference
		if dest.Channel != nil {
			channelRef = assets.NewChannelReference(dest.Channel.UUID(), dest.Channel.Name())
		}

		feedback_request := flows.NewFeedbackRequest(dest.URN.URN(), channelRef, a.StarRatingQuestion, a.CommentQuestion, a.SMSQuestion)
		logEvent(events.NewFeedbackRequestCreated(feedback_request))
	}

	return nil
}
