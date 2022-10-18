package events

import (
	"github.com/greatnonprofits-nfp/goflow/flows"
)

func init() {
	registerType(TypeDialogflowCalled, func() flows.Event { return &WebhookCalledEvent{} })
}

// TypeDialogflowCalled is the type for our lookup events
const TypeDialogflowCalled string = "dialogflow_called"

// NewDialogflowCalled returns a new dialogflow called event based on Webhook calls
func NewDialogflowCalled(call *flows.WebhookCall, status flows.CallStatus, resthook string) *WebhookCalledEvent {
	return &WebhookCalledEvent{
		baseEvent:   newBaseEvent(TypeDialogflowCalled),
		HTTPTrace:   flows.NewHTTPTrace(call.Trace, status),
		Resthook:    resthook,
		BodyIgnored: len(call.ResponseBody) > 0 && len(call.ResponseJSON) == 0, // i.e. there was a body but it couldn't be converted to JSON
	}
}
