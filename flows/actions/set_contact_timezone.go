package actions

import (
	"strings"
	"time"

	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/actions/modifiers"
	"github.com/greatnonprofits-nfp/goflow/flows/events"
)

func init() {
	RegisterType(TypeSetContactTimezone, func() flows.Action { return &SetContactTimezoneAction{} })
}

// TypeSetContactTimezone is the type for the set contact timezone action
const TypeSetContactTimezone string = "set_contact_timezone"

// SetContactTimezoneAction can be used to update the timezone of the contact. The timezone is a localizable
// template and white space is trimmed from the final value. An empty string clears the timezone.
// A [event:contact_timezone_changed] event will be created with the corresponding value.
//
//   {
//     "uuid": "8eebd020-1af5-431c-b943-aa670fc74da9",
//     "type": "set_contact_timezone",
//     "timezone": "Africa/Kigali"
//   }
//
// @action set_contact_timezone
type SetContactTimezoneAction struct {
	BaseAction
	universalAction

	Timezone string `json:"timezone"`
}

// NewSetContactTimezoneAction creates a new set timezone action
func NewSetContactTimezoneAction(uuid flows.ActionUUID, timezone string) *SetContactTimezoneAction {
	return &SetContactTimezoneAction{
		BaseAction: NewBaseAction(TypeSetContactTimezone, uuid),
		Timezone:   timezone,
	}
}

// Execute runs this action
func (a *SetContactTimezoneAction) Execute(run flows.FlowRun, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	if run.Contact() == nil {
		logEvent(events.NewErrorEventf("can't execute action in session without a contact"))
		return nil
	}

	timezone, err := run.EvaluateTemplate(a.Timezone)
	timezone = strings.TrimSpace(timezone)

	// if we received an error, log it
	if err != nil {
		logEvent(events.NewErrorEvent(err))
		return nil
	}

	// timezone must be empty or valid timezone name
	var tz *time.Location
	if timezone != "" {
		tz, err = time.LoadLocation(timezone)
		if err != nil {
			logEvent(events.NewErrorEventf("unrecognized timezone: '%s'", timezone))
			return nil
		}
	}

	a.applyModifier(run, modifiers.NewTimezoneModifier(tz), logModifier, logEvent)
	return nil
}

// Inspect inspects this object and any children
func (a *SetContactTimezoneAction) Inspect(inspect func(flows.Inspectable)) {
	inspect(a)
}

// EnumerateTemplates enumerates all expressions on this object and its children
func (a *SetContactTimezoneAction) EnumerateTemplates(include flows.TemplateIncluder) {
	include.String(&a.Timezone)
}
