package actions

import (
	"encoding/json"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/events"
	"github.com/pkg/errors"
	"net/http"
	"strings"
)

func init() {
	RegisterType(TypeCallGiftcard, func() flows.Action { return &CallGiftcardAction{} })
}

// TypeCallGiftcard is the type for the call giftcard action
const TypeCallGiftcard string = "call_giftcard"

// CallGiftcardAction can be used to call an external service. The body, header and url fields may be
// templates and will be evaluated at runtime. A [event:giftcard_called] event will be created based on
// the results of the HTTP call. If this action has a `result_name`, then addtionally it will create
// a new result with that name. If the lookup returned valid JSON, that will be accessible
// through `extra` on the result.
//
//   {
//     "uuid": "8eebd020-1af5-431c-b943-aa670fc74da9",
//     "type": "call_giftcard",
//     "giftcard_db": {"id": "demo_test_giftcard", "text": "Test Giftcard"},
//     "result_name": "giftcard"
//   }
//
// @action call_giftcard
type CallGiftcardAction struct {
	BaseAction
	onlineAction

	DB           map[string]string `json:"giftcard_db"`
	GiftcardType string            `json:"giftcard_type"`
	ResultName   string            `json:"result_name,omitempty"`
}

// NewCallGiftcardAction creates a new call giftcard action
func NewCallGiftcardAction(uuid flows.ActionUUID, giftcardDb map[string]string, giftcardType string, resultName string) *CallGiftcardAction {
	return &CallGiftcardAction{
		BaseAction:   NewBaseAction(TypeCallGiftcard, uuid),
		DB:           giftcardDb,
		GiftcardType: giftcardType,
		ResultName:   resultName,
	}
}

// Validate validates our action is valid
func (a *CallGiftcardAction) Validate() error {
	if a.DB["id"] == "" {
		return errors.Errorf("id is required on Giftcard DB")
	}

	return nil
}

// Execute runs this action
func (a *CallGiftcardAction) Execute(run flows.FlowRun, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	method := "POST"

	// substitute any variables in our url
	parseUrl := getEnv(envVarServerUrl, "http://localhost:9090/parse")
	var giftcardType string
	if a.GiftcardType == giftcardCheckType {
		giftcardType = "giftcards_remaining"
	} else {
		giftcardType = "giftcard"
	}
	url := parseUrl + "/functions/" + giftcardType

	if parseUrl == "" {
		logEvent(events.NewErrorEventf("Parse Server URL is an empty string, skipping"))
		return nil
	}

	if a.DB["id"] == "" {
		logEvent(events.NewErrorEventf("Parse Server DB is required, skipping"))
		return nil
	}

	contact_urn := run.Contact().PreferredURN()

	body := make(map[string]interface{})
	body["db"] = a.DB["id"]
	body["urn"] = contact_urn.URN().Path()

	b, _ := json.Marshal(body)

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
		logEvent(events.NewGiftcardCalledEvent(webhook))
		if a.ResultName != "" {
			a.saveWebhookResult(run, step, a.ResultName, webhook, logEvent)
		}
	}

	return nil
}

// Inspect inspects this object and any children
func (a *CallGiftcardAction) Inspect(inspect func(flows.Inspectable)) {
	inspect(a)
}

// EnumerateResults enumerates all potential results on this object
func (a *CallGiftcardAction) EnumerateResults(node flows.Node, include func(*flows.ResultInfo)) {
	if a.ResultName != "" {
		include(flows.NewResultInfo(a.ResultName, webhookCategories, node))
	}
}
