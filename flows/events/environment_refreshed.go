package events

import (
	"encoding/json"

	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/utils"
)

func init() {
	RegisterType(TypeEnvironmentRefreshed, func() flows.Event { return &EnvironmentRefreshedEvent{} })
}

// TypeEnvironmentRefreshed is the type of our environment changed event
const TypeEnvironmentRefreshed string = "environment_refreshed"

// EnvironmentRefreshedEvent events are sent by the caller to tell the engine to update the session environment.
//
//   {
//     "type": "environment_refreshed",
//     "created_on": "2006-01-02T15:04:05Z",
//     "environment": {
//       "date_format": "YYYY-MM-DD",
//       "time_format": "hh:mm",
//       "timezone": "Africa/Kigali",
//       "default_language": "eng",
//       "allowed_languages": ["eng", "fra"]
//     }
//   }
//
// @event environment_refreshed
type EnvironmentRefreshedEvent struct {
	BaseEvent

	Environment json.RawMessage `json:"environment"`
}

// NewEnvironmentRefreshedEvent creates a new environment changed event
func NewEnvironmentRefreshedEvent(env utils.Environment) *EnvironmentRefreshedEvent {
	marshalled, _ := json.Marshal(env)
	return &EnvironmentRefreshedEvent{
		BaseEvent:   NewBaseEvent(TypeEnvironmentRefreshed),
		Environment: marshalled,
	}
}

var _ flows.Event = (*EnvironmentRefreshedEvent)(nil)
