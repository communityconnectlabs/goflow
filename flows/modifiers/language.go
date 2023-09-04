package modifiers

import (
	"encoding/json"

	"github.com/nyaruka/gocommon/i18n"
	"github.com/nyaruka/goflow/assets"
	"github.com/nyaruka/goflow/envs"
	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/flows/events"
	"github.com/nyaruka/goflow/utils"
)

func init() {
	registerType(TypeLanguage, readLanguageModifier)
}

// TypeLanguage is the type of our language modifier
const TypeLanguage string = "language"

// LanguageModifier modifies the language of a contact
type LanguageModifier struct {
	baseModifier

	Language i18n.Language `json:"language"`
}

// NewLanguage creates a new language modifier
func NewLanguage(language i18n.Language) *LanguageModifier {
	return &LanguageModifier{
		baseModifier: newBaseModifier(TypeLanguage),
		Language:     language,
	}
}

// Apply applies this modification to the given contact
func (m *LanguageModifier) Apply(eng flows.Engine, env envs.Environment, sa flows.SessionAssets, contact *flows.Contact, log flows.EventCallback) bool {
	if contact.Language() != m.Language {
		contact.SetLanguage(m.Language)
		log(events.NewContactLanguageChanged(m.Language))
		return true
	}
	return false
}

var _ flows.Modifier = (*LanguageModifier)(nil)

//------------------------------------------------------------------------------------------
// JSON Encoding / Decoding
//------------------------------------------------------------------------------------------

func readLanguageModifier(assets flows.SessionAssets, data json.RawMessage, missing assets.MissingCallback) (flows.Modifier, error) {
	m := &LanguageModifier{}
	return m, utils.UnmarshalAndValidate(data, m)
}
