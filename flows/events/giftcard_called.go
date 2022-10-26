package events

import (
	"github.com/nyaruka/goflow/flows"
)

func init() {
	registerType(TypeGiftcardCalled, func() flows.Event { return &WebhookCalledEvent{} })
}

// TypeGiftcardCalled is the type for our lookup events
const TypeGiftcardCalled string = "giftcard_called"

// NewGiftcardCalled returns a new giftcard called event based on Webhook calls
func NewGiftcardCalled(call *flows.WebhookCall, status flows.CallStatus, resthook string) *WebhookCalledEvent {
	return &WebhookCalledEvent{
		baseEvent:   newBaseEvent(TypeGiftcardCalled),
		HTTPTrace:   flows.NewHTTPTrace(call.Trace, status),
		Resthook:    resthook,
		BodyIgnored: len(call.ResponseBody) > 0 && len(call.ResponseJSON) == 0, // i.e. there was a body but it couldn't be converted to JSON
	}
}
