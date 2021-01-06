package events

import (
	"time"

	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/utils"
)

func init() {
	registerType(TypeLookupCalled, func() flows.Event { return &WebhookCalledEvent{} })
}

// TypeLookupCalled is the type for our lookup events
const TypeLookupCalled string = "lookup_called"

// NewLookupCalled returns a new lookup called event based on Webhook calls
func NewLookupCalled(call *flows.WebhookCall, status flows.CallStatus, resthook string) *WebhookCalledEvent {
	statusCode := 0
	if call.Response != nil {
		statusCode = call.Response.StatusCode
	}
	return &WebhookCalledEvent{
		baseEvent:   newBaseEvent(TypeLookupCalled),
		URL:         call.Request.URL.String(),
		Status:      status,
		Request:     utils.TruncateEllipsis(string(call.RequestTrace), trimTracesTo),
		Response:    utils.TruncateEllipsis(string(call.ResponseTrace), trimTracesTo),
		ElapsedMS:   int((call.EndTime.Sub(call.StartTime)) / time.Millisecond),
		Resthook:    resthook,
		StatusCode:  statusCode,
		BodyIgnored: len(call.ResponseBody) > 0 && !call.ValidJSON,
	}
}
