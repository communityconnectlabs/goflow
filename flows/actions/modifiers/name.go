package modifiers

import (
	"encoding/json"

	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/envs"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/events"
	"github.com/greatnonprofits-nfp/goflow/utils"
)

func init() {
	registerType(TypeName, readNameModifier)
}

// TypeName is the type of our name modifier
const TypeName string = "name"

// NameModifier modifies the name of a contact
type NameModifier struct {
	baseModifier

	Name string `json:"name"`
}

// NewName creates a new name modifier
func NewName(name string) *NameModifier {
	return &NameModifier{
		baseModifier: newBaseModifier(TypeName),
		Name:         name,
	}
}

// Apply applies this modification to the given contact
func (m *NameModifier) Apply(env envs.Environment, assets flows.SessionAssets, contact *flows.Contact, log flows.EventCallback) {
	if contact.Name() != m.Name {
		// truncate value if necessary
		if len(m.Name) > env.MaxValueLength() {
			m.Name = m.Name[0:env.MaxValueLength()]
		}

		contact.SetName(m.Name)
		log(events.NewContactNameChanged(m.Name))
		m.reevaluateDynamicGroups(env, assets, contact, log)
	}
}

var _ flows.Modifier = (*NameModifier)(nil)

//------------------------------------------------------------------------------------------
// JSON Encoding / Decoding
//------------------------------------------------------------------------------------------

func readNameModifier(assets flows.SessionAssets, data json.RawMessage, missing assets.MissingCallback) (flows.Modifier, error) {
	m := &NameModifier{}
	return m, utils.UnmarshalAndValidate(data, m)
}
