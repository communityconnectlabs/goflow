package events

import (
	"time"

	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/utils"
)

func init() {
	registerType(TypeDialogflowCalled, func() flows.Event { return &WebhookCalledEvent{} })
}

// TypeDialogflowCalled is the type for our lookup events
const TypeDialogflowCalled string = "dialogflow_called"

// NewDialogflowCalled returns a new dialogflow called event based on Webhook calls
func NewDialogflowCalled(call *flows.WebhookCall, status flows.CallStatus, resthook string) *WebhookCalledEvent {
	statusCode := 0
	if call.Response != nil {
		statusCode = call.Response.StatusCode
	}
	return &WebhookCalledEvent{
		baseEvent:   newBaseEvent(TypeDialogflowCalled),
		URL:         call.Request.URL.String(),
		Status:      status,
		Request:     utils.TruncateEllipsis(string(call.RequestTrace), trimTracesTo),
		Response:    utils.TruncateEllipsis(string(call.ResponseTrace), trimTracesTo),
		ElapsedMS:   int((call.EndTime.Sub(call.StartTime)) / time.Millisecond),
		Resthook:    resthook,
		StatusCode:  statusCode,
		BodyIgnored: len(call.ResponseBody) > 0 && len(call.ResponseJSON) == 0,
	}
}
