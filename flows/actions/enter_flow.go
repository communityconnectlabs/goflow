package actions

import (
	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/events"

	"github.com/pkg/errors"
)

func init() {
	RegisterType(TypeEnterFlow, func() flows.Action { return &EnterFlowAction{} })
}

// TypeEnterFlow is the type for the enter flow action
const TypeEnterFlow string = "enter_flow"

// EnterFlowAction can be used to start a contact down another flow. The current flow will pause until the subflow exits or expires.
//
// A [event:flow_entered] event will be created to record that the flow was started.
//
//   {
//     "uuid": "8eebd020-1af5-431c-b943-aa670fc74da9",
//     "type": "enter_flow",
//     "flow": {"uuid": "b7cf0d83-f1c9-411c-96fd-c511a4cfa86d", "name": "Collect Language"},
//     "terminal": false
//   }
//
// @action enter_flow
type EnterFlowAction struct {
	BaseAction
	universalAction

	Flow     *assets.FlowReference `json:"flow" validate:"required"`
	Terminal bool                  `json:"terminal,omitempty"`
}

// NewEnterFlowAction creates a new start flow action
func NewEnterFlowAction(uuid flows.ActionUUID, flow *assets.FlowReference, terminal bool) *EnterFlowAction {
	return &EnterFlowAction{
		BaseAction: NewBaseAction(TypeEnterFlow, uuid),
		Flow:       flow,
		Terminal:   terminal,
	}
}

// Execute runs our action
func (a *EnterFlowAction) Execute(run flows.FlowRun, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	flow, err := run.Session().Assets().Flows().Get(a.Flow.UUID)

	// we ignore other missing asset types but a missing flow means we don't know how to route so we can't continue
	if err != nil {
		run.Exit(flows.RunStatusErrored)
		logEvent(events.NewFatalErrorEvent(err))
		return nil
	}

	if run.Session().Type() != "" && run.Session().Type() != flow.Type() {
		run.Exit(flows.RunStatusErrored)
		logEvent(events.NewFatalErrorEvent(errors.Errorf("can't enter %s of type %s from type %s", flow.Reference(), flow.Type(), run.Session().Type())))
		return nil
	}

	run.Session().PushFlow(flow, run, a.Terminal)
	logEvent(events.NewFlowEnteredEvent(a.Flow, run.UUID(), a.Terminal))
	return nil
}

// Inspect inspects this object and any children
func (a *EnterFlowAction) Inspect(inspect func(flows.Inspectable)) {
	inspect(a)
	flows.InspectReference(a.Flow, inspect)
}
