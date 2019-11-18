package events

import (
	"time"

	"github.com/greatnonprofits-nfp/goflow/flows"
)

func init() {
	RegisterType(TypeGiftcardCalled, func() flows.Event { return &WebhookCalledEvent{} })
}

// TypeGiftcardCalled is the type for our lookup events
const TypeGiftcardCalled string = "giftcard_called"

// NewGiftcardCalledEvent returns a new giftcard called event based on Webhook calls
func NewGiftcardCalledEvent(webhook *flows.WebhookCall) *WebhookCalledEvent {
	return &WebhookCalledEvent{
		BaseEvent:   NewBaseEvent(TypeGiftcardCalled),
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
