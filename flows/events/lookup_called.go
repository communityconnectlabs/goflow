package events

import (
	"github.com/nyaruka/goflow/flows"
)

func init() {
	registerType(TypeLookupCalled, func() flows.Event { return &WebhookCalledEvent{} })
}

// TypeLookupCalled is the type for our lookup events
const TypeLookupCalled string = "lookup_called"

// NewLookupCalled returns a new lookup called event based on Webhook calls
func NewLookupCalled(call *flows.WebhookCall, status flows.CallStatus, resthook string) *WebhookCalledEvent {
	extraction := ExtractionNone
	if len(call.ResponseBody) > 0 {
		if len(call.ResponseJSON) > 0 {
			if call.ResponseCleaned {
				extraction = ExtractionCleaned
			} else {
				extraction = ExtractionValid
			}
		} else {
			extraction = ExtractionIgnored
		}
	}

	return &WebhookCalledEvent{
		BaseEvent:  NewBaseEvent(TypeLookupCalled),
		HTTPTrace:  flows.NewHTTPTrace(call.Trace, status),
		Resthook:   resthook,
		Extraction: extraction,
	}
}
