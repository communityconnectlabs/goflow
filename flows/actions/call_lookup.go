package actions

import (
	"github.com/greatnonprofits-nfp/goflow/flows"
)

func init() {
	RegisterType(TypeCallLookup, func() flows.Action { return &CallLookupAction{} })
}

// TypeCallLookup is the type for the call webhook action
const TypeCallLookup string = "call_lookup"

// CallLookupAction can be used to call an external service. The body, header and url fields may be
// templates and will be evaluated at runtime. A [event:lookup_called] event will be created based on
// the results of the HTTP call. If this action has a `result_name`, then addtionally it will create
// a new result with that name. If the webhook returned valid JSON, that will be accessible
// through `extra` on the result.
//
//   {
//     "uuid": "8eebd020-1af5-431c-b943-aa670fc74da9",
//     "type": "call_lookup",
//     "collection": "test_lookup",
//     "rules": [{
//       "field": "name",
//       "rule": "equals_to",
//       "value": "Marcus",
//     }],
//     "result_name": "lookup"
//   }
//
// @action call_lookup
type CallLookupAction struct {
	BaseAction
	onlineAction

	Collection string       `json:"collection" validate:"required"`
	Rules      []LookupRule `json:"rules,omitempty" validate:"required"`
	ResultName string       `json:"result_name,omitempty"`
}

type LookupRule struct {
	Field string `json:"field"`
	Rule  string `json:"rule"`
	Value string `json:"value"`
}

// NewCallLookupAction creates a new call lookup action
func NewCallLookupAction(uuid flows.ActionUUID, collection string, rules []LookupRule, resultName string) *CallLookupAction {
	return &CallLookupAction{
		BaseAction: NewBaseAction(TypeCallWebhook, uuid),
		Collection: collection,
		Rules:      rules,
		ResultName: resultName,
	}
}

// Validate validates our action is valid
func (a *CallLookupAction) Validate() error {
	return nil
}

// Execute runs this action
func (a *CallLookupAction) Execute(run flows.FlowRun, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	return nil
}

// Inspect inspects this object and any children
func (a *CallLookupAction) Inspect(inspect func(flows.Inspectable)) {
	inspect(a)
}

// EnumerateResults enumerates all potential results on this object
func (a *CallLookupAction) EnumerateResults(node flows.Node, include func(*flows.ResultInfo)) {
	if a.ResultName != "" {
		include(flows.NewResultInfo(a.ResultName, webhookCategories, node))
	}
}
