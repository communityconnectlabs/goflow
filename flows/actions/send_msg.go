package actions

import (
	"github.com/nyaruka/gocommon/urns"
	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/envs"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/events"
	"github.com/greatnonprofits-nfp/goflow/utils/uuids"
	"fmt"
	"net/url"
	"net/http"
	"strings"
	"io/ioutil"
	"github.com/buger/jsonparser"
	"regexp"
)

func init() {
	registerType(TypeSendMsg, func() flows.Action { return &SendMsgAction{} })
}

// TypeSendMsg is the type for the send message action
const TypeSendMsg string = "send_msg"

// SendMsgAction can be used to reply to the current contact in a flow. The text field may contain templates. The action
// will attempt to find pairs of URNs and channels which can be used for sending. If it can't find such a pair, it will
// create a message without a channel or URN.
//
// A [event:msg_created] event will be created with the evaluated text.
//
//   {
//     "uuid": "8eebd020-1af5-431c-b943-aa670fc74da9",
//     "type": "send_msg",
//     "text": "Hi @contact.name, are you ready to complete today's survey?",
//     "attachments": [],
//     "all_urns": false,
//     "templating": {
//       "uuid": "32c2ead6-3fa3-4402-8e27-9cc718175c5a",
//       "template": {
//         "uuid": "3ce100b7-a734-4b4e-891b-350b1279ade2",
//         "name": "revive_issue"
//       },
//       "variables": ["@contact.name"]
//     },
//     "topic": "event"
//   }
//
// @action send_msg
type SendMsgAction struct {
	baseAction
	universalAction
	createMsgAction

	AllURNs    bool           `json:"all_urns,omitempty"`
	Templating *Templating    `json:"templating,omitempty" validate:"omitempty,dive"`
	Topic      flows.MsgTopic `json:"topic,omitempty" validate:"omitempty,msg_topic"`
}

// Templating represents the templating that should be used if possible
type Templating struct {
	UUID      uuids.UUID                `json:"uuid" validate:"required,uuid4"`
	Template  *assets.TemplateReference `json:"template" validate:"required"`
	Variables []string                  `json:"variables" engine:"localized,evaluated"`
}

// LocalizationUUID gets the UUID which identifies this object for localization
func (t *Templating) LocalizationUUID() uuids.UUID { return t.UUID }

// NewSendMsg creates a new send msg action
func NewSendMsg(uuid flows.ActionUUID, text string, attachments []string, quickReplies []string, allURNs bool) *SendMsgAction {
	return &SendMsgAction{
		baseAction: newBaseAction(TypeSendMsg, uuid),
		createMsgAction: createMsgAction{
			Text:         text,
			Attachments:  attachments,
			QuickReplies: quickReplies,
		},
		AllURNs: allURNs,
	}
}

// Execute runs this action
func (a *SendMsgAction) Execute(run flows.FlowRun, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	if run.Contact() == nil {
		logEvent(events.NewErrorf("can't execute action in session without a contact"))
		return nil
	}

	evaluatedText, evaluatedAttachments, evaluatedQuickReplies := a.evaluateMessage(run, nil, a.Text, a.Attachments, a.QuickReplies, logEvent)

	destinations := run.Contact().ResolveDestinations(a.AllURNs)

	sa := run.Session().Assets()

	orgLinks := run.Environment().Links()

	yoURLsHost := getEnv(envVarYoURLsHost, "")
	yoURLsLogin := getEnv(envVarYoURLsLogin, "")
	yoURLsPassword := getEnv(envVarYoURLsPassword, "")
	mailroomDomain := getEnv(envVarMailroomDomain, "")

	text := evaluatedText

	// Whether we don't have the YoURLs credentials, should be skipped
	if yoURLsHost != "" && yoURLsLogin != "" && yoURLsPassword != "" && mailroomDomain != "" {

		fmt.Println(yoURLsHost)
		fmt.Println(yoURLsLogin)
		fmt.Println(yoURLsPassword)

		// splitting the text as array for analyzing and replace if it's the case
		re := regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?!&//=]*)`)
		textSplitted := re.FindAllString(text, -1)

		fmt.Println(textSplitted)

		for i := range textSplitted {
			d := textSplitted[i]

			// checking if the text is a valid URL
			if !isValidURL(d) {
				continue
			}

			destUUID, destLink := findDestinationInLinks(d, orgLinks)

			if destUUID == "" || destLink == "" {
				continue
			}

			fmt.Println(destUUID)
			fmt.Println(destLink)

			fmt.Println(string(run.Contact().UUID()))

			if string(run.Contact().UUID()) != "" {
				yourlsURL := fmt.Sprintf("%s/yourls-api.php", yoURLsHost)
				handleURL := fmt.Sprintf("https://%s/link/handler/%s", mailroomDomain, destUUID)
				longURL := fmt.Sprintf("%s?contact=%s", handleURL, string(run.Contact().UUID()))

				// creating the payload
				payload := url.Values{}
				payload.Add("url", longURL)
				payload.Add("format", "json")
				payload.Add("action", "shorturl")
				payload.Add("username", yoURLsLogin)
				payload.Add("password", yoURLsPassword)

				// build our request
				method := "GET"
				yourlsURL = fmt.Sprintf("%s?%s", yourlsURL, payload.Encode())
				req, errReq := http.NewRequest(method, yourlsURL, strings.NewReader(""))
				if errReq != nil {
					continue
				}

				req.Header.Add("Content-Type", "multipart/form-data")

				resp, errHttp := http.DefaultClient.Do(req)
				if errHttp != nil {
					continue
				}
				content, errRead := ioutil.ReadAll(resp.Body)
				if errRead != nil {
					continue
				}

				// replacing the link for the YoURLs generated link
				shortLink, _ := jsonparser.GetString(content, "shorturl")
				text = strings.Replace(text, d, shortLink, -1)

				fmt.Println(text)

			}

		}

	}

	// create a new message for each URN+channel destination
	for _, dest := range destinations {
		var channelRef *assets.ChannelReference
		if dest.Channel != nil {
			channelRef = assets.NewChannelReference(dest.Channel.UUID(), dest.Channel.Name())
		}

		var templating *flows.MsgTemplating

		// do we have a template defined?
		if a.Templating != nil {
			translation := sa.Templates().FindTranslation(a.Templating.Template.UUID, channelRef, []envs.Language{run.Contact().Language(), run.Environment().DefaultLanguage()})
			if translation != nil {
				localizedVariables := run.GetTextArray(uuids.UUID(a.Templating.UUID), "variables", a.Templating.Variables)

				// evaluate our variables
				evaluatedVariables := make([]string, len(localizedVariables))
				for i, variable := range localizedVariables {
					sub, err := run.EvaluateTemplate(variable)
					if err != nil {
						logEvent(events.NewError(err))
					}
					evaluatedVariables[i] = sub
				}

				evaluatedText = translation.Substitute(evaluatedVariables)
				templating = flows.NewMsgTemplating(a.Templating.Template, translation.Language(), evaluatedVariables)
			}
		}

		msg := flows.NewMsgOut(dest.URN.URN(), channelRef, evaluatedText, evaluatedAttachments, evaluatedQuickReplies, templating, a.Topic)
		logEvent(events.NewMsgCreated(msg))
	}

	// if we couldn't find a destination, create a msg without a URN or channel and it's up to the caller
	// to handle that as they want
	if len(destinations) == 0 {
		msg := flows.NewMsgOut(urns.NilURN, nil, text, evaluatedAttachments, evaluatedQuickReplies, nil, flows.NilMsgTopic)
		logEvent(events.NewMsgCreated(msg))
	}

	return nil
}
