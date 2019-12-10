package events

import (
	"time"

	"github.com/greatnonprofits-nfp/goflow/flows"
)

func init() {
	RegisterType(TypeShortenURLCalled, func() flows.Event { return &WebhookCalledEvent{} })
}

// TypeShortenURLCalled is the type for our shorten url events
const TypeShortenURLCalled string = "shorten_url_called"

// NewShortenURLCalledEvent returns a new shorten url called event based on Webhook calls
func NewShortenURLCalledEvent(webhook *flows.WebhookCall) *WebhookCalledEvent {
	return &WebhookCalledEvent{
		BaseEvent:   NewBaseEvent(TypeShortenURLCalled),
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
