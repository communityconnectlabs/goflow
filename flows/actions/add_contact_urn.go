package actions

import (
	"strings"

	"github.com/nyaruka/gocommon/urns"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/actions/modifiers"
	"github.com/greatnonprofits-nfp/goflow/flows/events"

	"github.com/pkg/errors"
)

func init() {
	RegisterType(TypeAddContactURN, func() flows.Action { return &AddContactURNAction{} })
}

// TypeAddContactURN is our type for the add URN action
const TypeAddContactURN string = "add_contact_urn"

// AddContactURNAction can be used to add a URN to the current contact. A [event:contact_urns_changed] event
// will be created when this action is encountered.
//
//   {
//     "uuid": "8eebd020-1af5-431c-b943-aa670fc74da9",
//     "type": "add_contact_urn",
//     "scheme": "tel",
//     "path": "@results.phone_number.value"
//   }
//
// @action add_contact_urn
type AddContactURNAction struct {
	BaseAction
	universalAction

	Scheme string `json:"scheme" validate:"urnscheme"`
	Path   string `json:"path" validate:"required"`
}

// NewAddContactURNAction creates a new add URN action
func NewAddContactURNAction(uuid flows.ActionUUID, scheme string, path string) *AddContactURNAction {
	return &AddContactURNAction{
		BaseAction: NewBaseAction(TypeAddContactURN, uuid),
		Scheme:     scheme,
		Path:       path,
	}
}

// Execute runs the labeling action
func (a *AddContactURNAction) Execute(run flows.FlowRun, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	// only generate event if run has a contact
	contact := run.Contact()
	if contact == nil {
		logEvent(events.NewErrorEventf("can't execute action in session without a contact"))
		return nil
	}

	evaluatedPath, err := run.EvaluateTemplate(a.Path)

	// if we received an error, log it although it might just be a non-expression like foo@bar.com
	if err != nil {
		logEvent(events.NewErrorEvent(err))
	}

	evaluatedPath = strings.TrimSpace(evaluatedPath)
	if evaluatedPath == "" {
		logEvent(events.NewErrorEventf("can't add URN with empty path"))
		return nil
	}

	// if we don't have a valid URN, log error
	urn, err := urns.NewURNFromParts(a.Scheme, evaluatedPath, "", "")
	if err != nil {
		logEvent(events.NewErrorEvent(errors.Wrapf(err, "unable to add URN '%s:%s'", a.Scheme, evaluatedPath)))
		return nil
	}

	a.applyModifier(run, modifiers.NewURNModifier(urn, modifiers.URNAppend), logModifier, logEvent)
	return nil
}

// Inspect inspects this object and any children
func (a *AddContactURNAction) Inspect(inspect func(flows.Inspectable)) {
	inspect(a)
}

// EnumerateTemplates enumerates all expressions on this object and its children
func (a *AddContactURNAction) EnumerateTemplates(include flows.TemplateIncluder) {
	include.String(&a.Path)
}
