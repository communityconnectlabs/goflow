package actions

import (
	"encoding/json"
	"fmt"
	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/flows/events"
	"github.com/nyaruka/goflow/utils"
	"net/http"
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
func (a *VoiceCallStatusAction) Execute(run flows.Run, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	callSID := run.Session().Trigger().Connection().ExternalID()
	mailroomDomain := utils.GetEnv(utils.MailroomDomain, "example.com")
	channelUUID := run.Session().Contact().PreferredChannel().UUID()

	url := fmt.Sprintf("https://%s/mr/ivr/c/%s/voice-call-status?external_id=%s", mailroomDomain, channelUUID, callSID)

	// build our request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

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

		// whether status is empty, try 5x before goes to failure
		if status == "" {
			sleepFor := int(1 * time.Second)
			maxTries := 4

			for i := 0; i <= maxTries; i++ {
				time.Sleep(time.Duration(sleepFor))

				loopCall, callErr := svc.Call(run.Session(), req)
				status = voiceCallStatus(loopCall, callErr)

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
		"machine":             flows.CallStatusMachineEndOther,
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
