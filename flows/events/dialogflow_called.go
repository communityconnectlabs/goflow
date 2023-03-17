package events

import (
	"github.com/nyaruka/goflow/flows"
)

func init() {
	registerType(TypeDialogflowCalled, func() flows.Event { return &WebhookCalledEvent{} })
}

// TypeDialogflowCalled is the type for our lookup events
const TypeDialogflowCalled string = "dialogflow_called"

// NewDialogflowCalled returns a new dialogflow called event based on Webhook calls
func NewDialogflowCalled(call *flows.WebhookCall, status flows.CallStatus, resthook string) *WebhookCalledEvent {
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
		BaseEvent:  NewBaseEvent(TypeDialogflowCalled),
		HTTPTrace:  flows.NewHTTPTrace(call.Trace, status),
		Resthook:   resthook,
		Extraction: extraction,
	}
}
