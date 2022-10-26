package events

import (
	"github.com/nyaruka/goflow/flows"
)

func init() {
	registerType(TypeShortenURLCalled, func() flows.Event { return &WebhookCalledEvent{} })
}

// TypeShortenURLCalled is the type for our shorten url events
const TypeShortenURLCalled string = "shorten_url_called"

// NewShortenURLCalled returns a new shorten url called event based on Webhook calls
func NewShortenURLCalled(call *flows.WebhookCall, status flows.CallStatus, resthook string) *WebhookCalledEvent {
	return &WebhookCalledEvent{
		baseEvent:   newBaseEvent(TypeShortenURLCalled),
		HTTPTrace:   flows.NewHTTPTrace(call.Trace, status),
		Resthook:    resthook,
		BodyIgnored: len(call.ResponseBody) > 0 && len(call.ResponseJSON) == 0, // i.e. there was a body but it couldn't be converted to JSON
	}
}
