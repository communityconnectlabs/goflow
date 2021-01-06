package events

import (
	"time"

	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/utils"
)

func init() {
	registerType(TypeGiftcardCalled, func() flows.Event { return &WebhookCalledEvent{} })
}

// TypeGiftcardCalled is the type for our lookup events
const TypeGiftcardCalled string = "giftcard_called"

// NewGiftcardCalled returns a new giftcard called event based on Webhook calls
func NewGiftcardCalled(call *flows.WebhookCall, status flows.CallStatus, resthook string) *WebhookCalledEvent {
	statusCode := 0
	if call.Response != nil {
		statusCode = call.Response.StatusCode
	}
	return &WebhookCalledEvent{
		baseEvent:   newBaseEvent(TypeGiftcardCalled),
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
