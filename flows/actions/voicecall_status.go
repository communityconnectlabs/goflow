package actions

import (
	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/flows/events"
	"net/http"
	"strings"
	"fmt"
	"encoding/json"
	"time"
)

func init() {
	registerType(TypeVoiceCallStatus, func() flows.Action { return &VoiceCallStatusAction{} })
}

// TypeVoiceCallStatus is the type for the call lookup action
const TypeVoiceCallStatus string = "voicecall_status"

// VoiceCallStatusAction can be used to call the Twilio API to check the voice call status.
// A [event:voicecall_status] event will be created based on the results of the HTTP call.
//
//   {
//     "uuid": "8eebd020-1af5-431c-b943-aa670fc74da9",
//     "type": "voicecall_status",
//     "result_name": "voicecall_status"
//   }
//
// @action voicecall_status
type VoiceCallStatusAction struct {
	baseAction
	onlineAction

	ResultName string `json:"result_name,omitempty"`
}

// NewVoiceCallStatusAction creates a new call lookup action
func NewVoiceCallStatusAction(uuid flows.ActionUUID, resultName string) *VoiceCallStatusAction {
	return &VoiceCallStatusAction{
		baseAction: newBaseAction(TypeVoiceCallStatus, uuid),
		ResultName: resultName,
	}
}

// Validate validates our action is valid
func (a *VoiceCallStatusAction) Validate() error {
	return nil
}

// Execute runs this action
func (a *VoiceCallStatusAction) Execute(run flows.FlowRun, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	callSID := run.Session().Trigger().Connection().ExternalID()
	twilioCreds := run.Session().Trigger().Connection().TwilioCredentials()

	run.Session().Trigger().Connection().Channel().Type()

	credentials := strings.Split(twilioCreds, ":")

	// whether we don't have the credentials
	if len(credentials) != 2 {
		return nil
	}

	accountSID := credentials[0]
	accountToken := credentials[1]

	method := "GET"
	url := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Calls/%s.json", accountSID, callSID)
	body := ""

	// build our request
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return err
	}

	req.SetBasicAuth(accountSID, accountToken)

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
		status := voiceCallStatus(call, err)

		// whether status is empty, try 3x before goes to failure
		if status == "" {
			sleepFor := int(5 * time.Second)
			maxTries := 2

			for i := 0; i <= maxTries; i++ {
				time.Sleep(time.Duration(i * sleepFor))

				call, _ := svc.Call(run.Session(), req)
				status = voiceCallStatus(call, err)

				if status != "" {
					break
				}
			}
		}

		if status == "" {
			status = flows.CallStatusConnectionError
		}

		if a.ResultName != "" {
			a.saveVoiceCallResult(run, step, a.ResultName, call, status, logEvent)
		}
	}

	return nil

}

// Results enumerates any results generated by this flow object
func (a *VoiceCallStatusAction) Results(include func(*flows.ResultInfo)) {
	if a.ResultName != "" {
		include(flows.NewResultInfo(a.ResultName, voiceCallCategories))
	}
}

// determines the webhook status from the HTTP status code
func voiceCallStatus(call *flows.WebhookCall, err error) flows.CallStatus {
	if call.Response == nil || err != nil {
		return flows.CallStatusConnectionError
	}

	var result map[string]interface{}
	err = json.Unmarshal(call.ResponseBody, &result)
	if err != nil {
		return flows.CallStatusResponseError
	}

	if result["answered_by"] == nil {
		return ""
	}

	voiceStatusCategories := map[string]flows.CallStatus{
		"human":               flows.CallStatusVoiceHuman,
		"unknown":             flows.CallStatusVoiceUnknown,
		"machine_end_beep":    flows.CallStatusMachineEndBeep,
		"machine_end_silence": flows.CallStatusMachineEndSilence,
		"machine_end_other":   flows.CallStatusMachineEndOther,
	}

	answeredBy := result["answered_by"].(string)

	if call.Response.StatusCode/100 == 2 {
		return voiceStatusCategories[answeredBy]
	}

	return ""
}
