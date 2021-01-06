package events

import (
	"time"

	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/utils"
)

func init() {
	registerType(TypeShortenURLCalled, func() flows.Event { return &WebhookCalledEvent{} })
}

// TypeShortenURLCalled is the type for our shorten url events
const TypeShortenURLCalled string = "shorten_url_called"

// NewShortenURLCalled returns a new shorten url called event based on Webhook calls
func NewShortenURLCalled(call *flows.WebhookCall, status flows.CallStatus, resthook string) *WebhookCalledEvent {
	statusCode := 0
	if call.Response != nil {
		statusCode = call.Response.StatusCode
	}
	return &WebhookCalledEvent{
		baseEvent:   newBaseEvent(TypeShortenURLCalled),
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
