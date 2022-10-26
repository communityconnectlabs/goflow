package actions

import (
	"fmt"
	"github.com/nyaruka/goflow/assets"
	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/flows/events"
	"github.com/nyaruka/goflow/services/classification/dialogflowcl"
	"github.com/nyaruka/gocommon/jsonx"
	"github.com/pkg/errors"
	"golang.org/x/text/language"
	dialogflowpb "google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
)

func init() {
	registerType(TypeCallDialogflow, func() flows.Action { return &CallCallDialogflowAction{} })
}

// TypeCallDialogflow is the type for the call dialogflow action
const TypeCallDialogflow string = "call_dialogflow"

// CallCallDialogflowAction can be used to classify the intent and entities from a given input using an NLU classifier. It always
// saves a result indicating whether the classification was successful, skipped or failed, and what the extracted intents
// and entities were.
//
//   {
//     "uuid": "8eebd020-1af5-431c-b943-aa670fc74da9",
//     "type": "call_dialogflow",
//     "dialogflow_db": {
//       "id": "72a1f5df-49f9-45df-94c9-d86f7ea064e5",
//       "text": "Agent Name"
//     },
//     "question_src": "hi",
//     "result_name": "dialogflow_result"
//   }
//
// @action call_dialogflow
type CallCallDialogflowAction struct {
	baseAction
	onlineAction
	DB          map[string]string `json:"dialogflow_db"`
	QuestionSrc string            `json:"question_src" validate:"required"`
	ResultName  string            `json:"result_name" validate:"required"`
}

// Validate validates our action is valid
func (a *CallCallDialogflowAction) Validate() error {
	if a.DB["id"] == "" {
		return errors.Errorf("id is required on Dialogflow DB")
	}

	return nil
}

func (a *CallCallDialogflowAction) Execute(run flows.FlowRun, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	classifiers := run.Session().Assets().Classifiers()
	classifier := classifiers.Get(assets.ClassifierUUID(a.DB["id"]))
	if classifier == nil {
		// end execution when no classifier is found
		return nil
	}

	config := classifier.Classifier.Config()
	configStr, err := jsonx.Marshal(config)

	if err != nil {
		logEvent(events.NewError(err))
	}
	// substitute any variables in our input
	input, err := run.EvaluateTemplate(a.QuestionSrc)
	if err != nil {
		logEvent(events.NewError(err))
	}
	contact := run.Contact()
	languageCode := string(contact.Language())
	contactId := string(contact.UUID())
	ISO1Tag, err := language.Parse(languageCode)
	if err != nil {
		return err
	}
	languageCode = ISO1Tag.String()
	projectID := config["project_id"]
	resp, err := a.DetectIntentText(projectID, languageCode, input, contactId, configStr)

	if err != nil {
		logEvent(events.NewError(err))
	}
	if resp != nil {
		a.saveSuccess(run, step, input, resp, logEvent)
	}
	return nil
}

func (a *CallCallDialogflowAction) saveSuccess(run flows.FlowRun, step flows.Step, input string, response *dialogflowpb.DetectIntentResponse, logEvent flows.EventCallback) {
	queryResult := response.GetQueryResult()
	value := queryResult.GetFulfillmentText()

	extra, _ := jsonx.Marshal(queryResult)

	a.saveResult(run, step, a.ResultName, value, CategorySuccess, "", input, extra, logEvent)
}

func (a *CallCallDialogflowAction) DetectIntentText(projectID, languageCode, text, contactId string, config []byte) (*dialogflowpb.DetectIntentResponse, error) {
	if config == nil {
		return nil, errors.New("service account credential is required to run dialogflow")
	}

	if projectID == "" {
		return nil, errors.New(fmt.Sprintf("Received empty project (%s)", projectID))
	}

	c := dialogflowcl.Client{CredentialJSON: config, ProjectID: projectID}
	return c.DetectIntent(text, languageCode, contactId)
}
