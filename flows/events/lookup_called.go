package events

import (
	"github.com/greatnonprofits-nfp/goflow/flows"
)

func init() {
	registerType(TypeLookupCalled, func() flows.Event { return &WebhookCalledEvent{} })
}

// TypeLookupCalled is the type for our lookup events
const TypeLookupCalled string = "lookup_called"

// NewLookupCalled returns a new lookup called event based on Webhook calls
func NewLookupCalled(call *flows.WebhookCall, status flows.CallStatus, resthook string) *WebhookCalledEvent {
	return &WebhookCalledEvent{
		baseEvent:   newBaseEvent(TypeLookupCalled),
		HTTPTrace:   flows.NewHTTPTrace(call.Trace, status),
		Resthook:    resthook,
		BodyIgnored: len(call.ResponseBody) > 0 && len(call.ResponseJSON) == 0, // i.e. there was a body but it couldn't be converted to JSON
	}
}
