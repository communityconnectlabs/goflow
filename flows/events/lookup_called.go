package events

import (
	"time"

	"github.com/greatnonprofits-nfp/goflow/flows"
)

func init() {
	RegisterType(TypeLookupCalled, func() flows.Event { return &WebhookCalledEvent{} })
}

// TypeLookupCalled is the type for our lookup events
const TypeLookupCalled string = "lookup_called"

// NewLookupCalledEvent returns a new lookup called event based on Webhook calls
func NewLookupCalledEvent(webhook *flows.WebhookCall) *WebhookCalledEvent {
	return &WebhookCalledEvent{
		BaseEvent:   NewBaseEvent(TypeLookupCalled),
		URL:         webhook.URL(),
		Resthook:    webhook.Resthook(),
		Status:      webhook.Status(),
		StatusCode:  webhook.StatusCode(),
		ElapsedMS:   int(webhook.TimeTaken() / time.Millisecond),
		Request:     webhook.Request(),
		Response:    webhook.Response(),
		BodyIgnored: webhook.BodyIgnored(),
	}
}
