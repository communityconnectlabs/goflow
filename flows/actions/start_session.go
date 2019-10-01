package actions

import (
	"encoding/json"

	"github.com/nyaruka/gocommon/urns"
	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/events"
)

func init() {
	RegisterType(TypeStartSession, func() flows.Action { return &StartSessionAction{} })
}

// TypeStartSession is the type for the start session action
const TypeStartSession string = "start_session"

// StartSessionAction can be used to trigger sessions for other contacts and groups. A [event:session_triggered] event
// will be created and it's the responsibility of the caller to act on that by initiating a new session with the flow engine.
//
//   {
//     "uuid": "8eebd020-1af5-431c-b943-aa670fc74da9",
//     "type": "start_session",
//     "flow": {"uuid": "b7cf0d83-f1c9-411c-96fd-c511a4cfa86d", "name": "Registration"},
//     "groups": [
//       {"uuid": "1e1ce1e1-9288-4504-869e-022d1003c72a", "name": "Customers"}
//     ]
//   }
//
// @action start_session
type StartSessionAction struct {
	BaseAction
	onlineAction
	otherContactsAction

	Flow          *assets.FlowReference `json:"flow" validate:"required"`
	CreateContact bool                  `json:"create_contact,omitempty"`
}

// NewStartSessionAction creates a new start session action
func NewStartSessionAction(uuid flows.ActionUUID, flow *assets.FlowReference, urns []urns.URN, contacts []*flows.ContactReference, groups []*assets.GroupReference, legacyVars []string, createContact bool) *StartSessionAction {
	return &StartSessionAction{
		BaseAction: NewBaseAction(TypeStartSession, uuid),
		otherContactsAction: otherContactsAction{
			URNs:       urns,
			Contacts:   contacts,
			Groups:     groups,
			LegacyVars: legacyVars,
		},
		Flow:          flow,
		CreateContact: createContact,
	}
}

// Execute runs our action
func (a *StartSessionAction) Execute(run flows.FlowRun, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	urnList, contactRefs, groupRefs, err := a.resolveRecipients(run, a.URNs, a.Contacts, a.Groups, a.LegacyVars, logEvent)
	if err != nil {
		return err
	}

	runSnapshot, err := json.Marshal(run.Snapshot())
	if err != nil {
		return err
	}

	// if we have any recipients, log an event
	if len(urnList) > 0 || len(contactRefs) > 0 || len(groupRefs) > 0 || a.CreateContact {
		logEvent(events.NewSessionTriggeredEvent(a.Flow, urnList, contactRefs, groupRefs, a.CreateContact, runSnapshot))
	}
	return nil
}

// Inspect inspects this object and any children
func (a *StartSessionAction) Inspect(inspect func(flows.Inspectable)) {
	inspect(a)
	flows.InspectReference(a.Flow, inspect)

	for _, g := range a.Groups {
		flows.InspectReference(g, inspect)
	}
	for _, c := range a.Contacts {
		flows.InspectReference(c, inspect)
	}
}

// EnumerateTemplates enumerates all expressions on this object and its children
func (a *StartSessionAction) EnumerateTemplates(include flows.TemplateIncluder) {
	include.Slice(a.LegacyVars)
}
