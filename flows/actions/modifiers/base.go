package modifiers

import (
	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/flows/events"
	"github.com/nyaruka/goflow/utils"
)

var registeredTypes = map[string](func() Modifier){}

// RegisterType registers a new type of modifier
func RegisterType(name string, initFunc func() Modifier) {
	registeredTypes[name] = initFunc
}

// Modifier is something which can modify a contact
type Modifier interface {
	// Apply applies this modification to the given contact
	Apply(utils.Environment, flows.SessionAssets, *flows.Contact, func(flows.Event))
}

// the base of all modifier types
type baseModifier struct {
	Type_ string `json:"type" validate:"required"`
}

func newBaseModifier(typeName string) baseModifier {
	return baseModifier{Type_: typeName}
}

// Type returns the type of this modifier
func (m *baseModifier) Type() string { return m.Type_ }

// helper to re-evaluate dynamic groups and log any changes to membership
func (m *baseModifier) reevaluateDynamicGroups(env utils.Environment, assets flows.SessionAssets, contact *flows.Contact, log func(flows.Event)) {
	added, removed, errors := contact.ReevaluateDynamicGroups(env, assets.Groups())

	// add error event for each group we couldn't re-evaluate
	for _, err := range errors {
		log(events.NewErrorEvent(err))
	}

	// add groups changed event for the groups we were added/removed to/from
	if len(added) > 0 || len(removed) > 0 {
		log(events.NewContactGroupsChangedEvent(added, removed))
	}
}
