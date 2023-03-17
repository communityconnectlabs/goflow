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
		BaseEvent:  NewBaseEvent(TypeShortenURLCalled),
		HTTPTrace:  flows.NewHTTPTrace(call.Trace, status),
		Resthook:   resthook,
		Extraction: extraction,
	}
}
