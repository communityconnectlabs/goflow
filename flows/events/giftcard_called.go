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
		BaseEvent:  NewBaseEvent(TypeGiftcardCalled),
		HTTPTrace:  flows.NewHTTPTrace(call.Trace, status),
		Resthook:   resthook,
		Extraction: extraction,
	}
}
