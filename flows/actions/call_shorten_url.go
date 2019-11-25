package actions

import (
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/events"
	"net/http"
	"strings"
)

func init() {
	RegisterType(TypeCallShortenURL, func() flows.Action { return &CallShortenURLAction{} })
}

// TypeCallLookup is the type for the call lookup action
const TypeCallShortenURL string = "call_shorten_url"

// CallShortenURLAction can be used to call the Firebase Dynamic URL API to generate shorten URLs.
// A [event:shorten_url_called] event will be created based on the results of the HTTP call.
// If this action has a `result_name`, then addtionally it will create
// a new result with that name. If the lookup returned valid JSON, that will be accessible
// through `extra` on the result.
//
//   {
//     "uuid": "8eebd020-1af5-431c-b943-aa670fc74da9",
//     "type": "call_shorten_url",
//     "shorten_url": {"id": "8eebd020-1af5-431c-b943-aa670fc74dc1", "text": "CCL Website"},
//     "result_name": "shorten_url"
//   }
//
// @action call_shorten_url
type CallShortenURLAction struct {
	BaseAction
	onlineAction

	ShortenURL map[string]string `json:"shorten_url"`
	ResultName string            `json:"result_name,omitempty"`
}

// NewCallShortenURLAction creates a new call lookup action
func NewCallShortenURLAction(uuid flows.ActionUUID, shortenURL map[string]string, resultName string) *CallShortenURLAction {
	return &CallShortenURLAction{
		BaseAction: NewBaseAction(TypeCallShortenURL, uuid),
		ShortenURL: shortenURL,
		ResultName: resultName,
	}
}

// Validate validates our action is valid
func (a *CallShortenURLAction) Validate() error {
	return nil
}

// Execute runs this action
func (a *CallShortenURLAction) Execute(run flows.FlowRun, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	// fake parameters
	method := "GET"
	url := getEnv(envVarShortenURLPing, "https://communityconnectlabs.com")
	body := ""

	// build our fake request
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return err
	}

	webhook, err := flows.MakeWebhookCall(run.Session(), req, "")

	if err != nil {
		logEvent(events.NewErrorEvent(err))
	} else {
		logEvent(events.NewShortenURLCalledEvent(webhook))
		if a.ResultName != "" {
			a.saveWebhookResult(run, step, a.ResultName, webhook, logEvent)
		}
	}

	return nil
}

// Inspect inspects this object and any children
func (a *CallShortenURLAction) Inspect(inspect func(flows.Inspectable)) {
	inspect(a)
}

// EnumerateResults enumerates all potential results on this object
func (a *CallShortenURLAction) EnumerateResults(node flows.Node, include func(*flows.ResultInfo)) {
	if a.ResultName != "" {
		include(flows.NewResultInfo(a.ResultName, webhookCategories, node))
	}
}
