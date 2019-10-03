package actions

import (
	"encoding/json"
	"fmt"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/events"
	"net/http"
	"os"
	"strings"
)

func init() {
	RegisterType(TypeCallLookup, func() flows.Action { return &CallLookupAction{} })
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
	BaseAction
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

const (
	xParseApplicationId = "X-Parse-Application-Id"
	xParseMasterKey     = "X-Parse-Master-Key"
	envVarAppId         = "MAILROOM_PARSE_SERVER_APP_ID"
	envVarMasterKey     = "MAILROOM_PARSE_SERVER_MASTER_KEY"
	envVarServerUrl     = "MAILROOM_PARSE_SERVER_URL"
)

// NewCallLookupAction creates a new call lookup action
func NewCallLookupAction(uuid flows.ActionUUID, lookupDb map[string]string, lookupQueries []LookupQuery, resultName string) *CallLookupAction {
	return &CallLookupAction{
		BaseAction: NewBaseAction(TypeCallLookup, uuid),
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
func (a *CallLookupAction) Execute(run flows.FlowRun, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	method := "POST"

	// substitute any variables in our url
	parseUrl := getEnv(envVarServerUrl, "http://localhost:9090/parse")
	url := parseUrl + "/functions/lookup"

	if parseUrl == "" {
		logEvent(events.NewErrorEventf("Parse Server URL is an empty string, skipping"))
		return nil
	}

	queries := make([]map[string]interface{}, 0, 0)

	// substitute any value variables
	for item := range a.Queries {
		queryValue, err := run.EvaluateTemplate(a.Queries[item].Value)
		if err != nil {
			logEvent(events.NewErrorEvent(err))
		}
		var newQuery = make(map[string]interface{})
		newQuery["field"] = a.Queries[item].Field
		newQuery["rule"] = a.Queries[item].Rule
		newQuery["value"] = queryValue
		queries = append(queries, newQuery)
	}

	if a.DB["id"] == "" {
		logEvent(events.NewErrorEventf("Parse Server DB is required, skipping"))
		return nil
	}

	body := make(map[string]interface{})
	body["queries"] = queries
	body["db"] = a.DB["id"]
	body["flow_step"] = true

	b, _ := json.Marshal(body)

	// TODO Remove those lines
	fmt.Println(a.Queries)
	fmt.Println(queries)

	// build our request
	req, err := http.NewRequest(method, url, strings.NewReader(string(b)))
	if err != nil {
		return err
	}

	appId := getEnv(envVarAppId, "myAppId")
	masterKey := getEnv(envVarMasterKey, "myMasterKey")

	// add the custom headers, substituting any template vars
	req.Header.Add(xParseApplicationId, appId)
	req.Header.Add(xParseMasterKey, masterKey)
	req.Header.Add("Content-Type", "application/json")

	webhook, err := flows.MakeWebhookCall(run.Session(), req, "")

	if err != nil {
		logEvent(events.NewErrorEvent(err))
	} else {
		logEvent(events.NewLookupCalledEvent(webhook))
		if a.ResultName != "" {
			a.saveWebhookResult(run, step, a.ResultName, webhook, logEvent)
		}
	}

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

func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
