package triggers

import (
	"encoding/json"

	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/excellent/types"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/utils"
)

func init() {
	RegisterType(TypeManual, readManualTrigger)
}

// TypeManual is the type for manually triggered sessions
const TypeManual string = "manual"

// ManualTrigger is used when a session was triggered manually by a user
//
//   {
//     "type": "manual",
//     "flow": {"uuid": "50c3706e-fedb-42c0-8eab-dda3335714b7", "name": "Registration"},
//     "contact": {
//       "uuid": "9f7ede93-4b16-4692-80ad-b7dc54a1cd81",
//       "name": "Bob",
//       "created_on": "2018-01-01T12:00:00.000000Z"
//     },
//     "triggered_on": "2000-01-01T00:00:00.000000000-00:00"
//   }
//
// @trigger manual
type ManualTrigger struct {
	baseTrigger
}

// NewManualTrigger creates a new manual trigger
func NewManualTrigger(env utils.Environment, flow *assets.FlowReference, contact *flows.Contact, params types.XValue) flows.Trigger {
	return &ManualTrigger{
		baseTrigger: newBaseTrigger(TypeManual, env, flow, contact, nil, params),
	}
}

// NewManualVoiceTrigger creates a new manual trigger with a channel connection for voice
func NewManualVoiceTrigger(env utils.Environment, flow *assets.FlowReference, contact *flows.Contact, connection *flows.Connection, params types.XValue) flows.Trigger {
	return &ManualTrigger{
		baseTrigger: newBaseTrigger(TypeManual, env, flow, contact, connection, params),
	}
}

var _ flows.Trigger = (*ManualTrigger)(nil)

//------------------------------------------------------------------------------------------
// JSON Encoding / Decoding
//------------------------------------------------------------------------------------------

func readManualTrigger(sessionAssets flows.SessionAssets, data json.RawMessage, missing assets.MissingCallback) (flows.Trigger, error) {
	e := &baseTriggerEnvelope{}
	if err := utils.UnmarshalAndValidate(data, e); err != nil {
		return nil, err
	}

	t := &ManualTrigger{}

	if err := t.unmarshal(sessionAssets, e, missing); err != nil {
		return nil, err
	}

	return t, nil
}

// MarshalJSON marshals this trigger into JSON
func (t *ManualTrigger) MarshalJSON() ([]byte, error) {
	e := &baseTriggerEnvelope{}

	if err := t.marshal(e); err != nil {
		return nil, err
	}

	return json.Marshal(e)
}
