package actions

import (
	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/events"
)

func init() {
	RegisterType(TypeAddInputLabels, func() flows.Action { return &AddInputLabelsAction{} })
}

// TypeAddInputLabels is the type for the add label action
const TypeAddInputLabels string = "add_input_labels"

// AddInputLabelsAction can be used to add labels to the last user input on a flow. An [event:input_labels_added] event
// will be created with the labels added when this action is encountered. If there is
// no user input at that point then this action will be ignored.
//
//   {
//     "uuid": "8eebd020-1af5-431c-b943-aa670fc74da9",
//     "type": "add_input_labels",
//     "labels": [{
//       "uuid": "3f65d88a-95dc-4140-9451-943e94e06fea",
//       "name": "Spam"
//     }]
//   }
//
// @action add_input_labels
type AddInputLabelsAction struct {
	BaseAction
	universalAction

	Labels []*assets.LabelReference `json:"labels" validate:"required,dive"`
}

// NewAddInputLabelsAction creates a new add labels action
func NewAddInputLabelsAction(uuid flows.ActionUUID, labels []*assets.LabelReference) *AddInputLabelsAction {
	return &AddInputLabelsAction{
		BaseAction: NewBaseAction(TypeAddInputLabels, uuid),
		Labels:     labels,
	}
}

// Execute runs the labeling action
func (a *AddInputLabelsAction) Execute(run flows.FlowRun, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	// log error if we don't have any input that could be labeled
	input := run.Session().Input()
	if input == nil {
		logEvent(events.NewErrorEventf("can't execute action in session without input"))
		return nil
	}

	labels, err := a.resolveLabels(run, a.Labels, logEvent)
	if err != nil {
		return err
	}

	if len(labels) > 0 {
		logEvent(events.NewInputLabelsAddedEvent(input.UUID(), labels))
	}

	return nil
}

// Inspect inspects this object and any children
func (a *AddInputLabelsAction) Inspect(inspect func(flows.Inspectable)) {
	inspect(a)

	for _, l := range a.Labels {
		flows.InspectReference(l, inspect)
	}
}
