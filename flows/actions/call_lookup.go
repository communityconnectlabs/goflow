package actions

import (
	"encoding/json"
	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/flows/events"
	"net/http"
	"strings"
	"github.com/nyaruka/goflow/utils"
)

func init() {
	registerType(TypeCallLookup, func() flows.Action { return &CallLookupAction{} })
}

// TypeCallLookup is the type for the call lookup action
const TypeCallLookup string = "call_lookup"

// CallLookupAction can be used to call an external service. The body, header and url fields may be
// templates and will be evaluated at runtime. A [event:lookup_called] event will be created based on
// the results of the HTTP call. If this action has a `result_name`, then addtionally it will create
// a new result with that name. If the lookup returned valid JSON, that will be accessible
// through `extra` on the result.
//
//   {
//     "uuid": "8eebd020-1af5-431c-b943-aa670fc74da9",
//     "type": "call_lookup",
//     "lookup_db": {"id": "demo_test_lookup", "text": "Test Lookup"},
//     "lookup_queries": [{
//     		"field": {"id": "name", "text": "name", "type": "String"},
//     		"rule": {"type": "equals", "verbose_name": "equals"},
//     		"value": "Marcus"
//     }],
//     "result_name": "lookup"
//   }
//
// @action call_lookup
type CallLookupAction struct {
	baseAction
	onlineAction

	DB         map[string]string `json:"lookup_db"`
	Queries    []LookupQuery     `json:"lookup_queries"`
	ResultName string            `json:"result_name,omitempty"`
}

type LookupQuery struct {
	Field map[string]string `json:"field"`
	Rule  map[string]string `json:"rule"`
	Value string            `json:"value"`
}

// NewCallLookupAction creates a new call lookup action
func NewCallLookupAction(uuid flows.ActionUUID, lookupDb map[string]string, lookupQueries []LookupQuery, resultName string) *CallLookupAction {
	return &CallLookupAction{
		baseAction: newBaseAction(TypeCallLookup, uuid),
		DB:         lookupDb,
		Queries:    lookupQueries,
		ResultName: resultName,
	}
}

// Validate validates our action is valid
func (a *CallLookupAction) Validate() error {
	return nil
}

// Execute runs this action
func (a *CallLookupAction) Execute(run flows.Run, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	method := "POST"

	// substitute any variables in our url
	parseUrl := utils.GetEnv(utils.ParseServerUrl, "http://localhost:9090/parse")
	url := parseUrl + "/functions/lookup"

	if parseUrl == "" {
		logEvent(events.NewErrorf("Parse Server URL is an empty string, skipping"))
		return nil
	}

	queries := make([]map[string]interface{}, 0, 0)

	// substitute any value variables
	for item := range a.Queries {
		queryValue, err := run.EvaluateTemplate(a.Queries[item].Value)
		if err != nil {
			logEvent(events.NewError(err))
		}
		var newQuery = make(map[string]interface{})
		newQuery["field"] = a.Queries[item].Field
		newQuery["rule"] = a.Queries[item].Rule
		newQuery["value"] = queryValue
		queries = append(queries, newQuery)
	}

	if a.DB["id"] == "" {
		logEvent(events.NewErrorf("Parse Server DB is required, skipping"))
		return nil
	}

	body := make(map[string]interface{})
	body["queries"] = queries
	body["db"] = a.DB["id"]
	body["flow_step"] = true

	b, _ := json.Marshal(body)

	return a.call(run, step, url, method, string(b), logEvent)
}

// Execute runs this action
func (a *CallLookupAction) call(run flows.Run, step flows.Step, url, method, body string, logEvent flows.EventCallback) error {
	// build our request
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return err
	}

	appId := utils.GetEnv(utils.ParseAppId, "myAppId")
	masterKey := utils.GetEnv(utils.ParseMasterKey, "myMasterKey")

	// add the custom headers, substituting any template vars
	req.Header.Add(utils.XParseApplicationId, appId)
	req.Header.Add(utils.XParseMasterKey, masterKey)
	req.Header.Add("Content-Type", "application/json")

	svc, err := run.Session().Engine().Services().Webhook(run.Session())
	if err != nil {
		logEvent(events.NewError(err))
		return nil
	}

	call, err := svc.Call(run.Session(), req)

	if err != nil {
		logEvent(events.NewError(err))
	}
	if call != nil {
		a.updateWebhook(run, call)

		status := callStatus(call, err, false)

		if a.ResultName != "" {
			a.saveWebhookResult(run, step, a.ResultName, call, status, logEvent)
		}
	}

	return nil
}

// Results enumerates any results generated by this flow object
func (a *CallLookupAction) Results(include func(*flows.ResultInfo)) {
	if a.ResultName != "" {
		include(flows.NewResultInfo(a.ResultName, webhookCategories))
	}
}
