package actions

import (
	"github.com/nyaruka/gocommon/urns"
	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/envs"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/events"
    "regexp"
    "strings"
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
//       "template": {
//         "uuid": "3ce100b7-a734-4b4e-891b-350b1279ade2",
//         "name": "revive_issue"
//       },
//       "variables": ["@contact.name"]
//     }
//   }
//
// @action send_msg
type SendMsgAction struct {
	baseAction
	universalAction
	createMsgAction

	AllURNs    bool        `json:"all_urns,omitempty"`
	Templating *Templating `json:"templating,omitempty" validate:"omitempty,dive"`
}

// Templating represents the templating that should be used if possible
type Templating struct {
	Template  *assets.TemplateReference `json:"template" validate:"required"`
	Variables []string                  `json:"variables" engine:"evaluated"`
}

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

	// create a new message for each URN+channel destination
	for _, dest := range destinations {
		var channelRef *assets.ChannelReference
		if dest.Channel != nil {
			channelRef = assets.NewChannelReference(dest.Channel.UUID(), dest.Channel.Name())
		}

		// channel uuid defined on RapidPro to be used on simulator
		simulatorChannelUUID := getEnv("MAILROOM_SIMULATOR_CHANNEL_UUID", "440099cf-200c-4d45-a8e7-4a564f4a0e8b")

		// making the replacing process for fake links if it is from the simulador
		if string(dest.Channel.UUID()) == simulatorChannelUUID {
			re := regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?!&//=]*)`)
			textSplitted := re.FindAllString(evaluatedText, -1)
			for i := range textSplitted {
				link := textSplitted[i]
				evaluatedText = strings.Replace(evaluatedText, link, "https://ccl.trackable.link", -1)
			}
		}

		var templating *flows.MsgTemplating

		// do we have a template defined?
		if a.Templating != nil {
			translation := sa.Templates().FindTranslation(a.Templating.Template.UUID, channelRef, []envs.Language{run.Contact().Language(), run.Environment().DefaultLanguage()})
			if translation != nil {
				// evaluate our variables
				templateVariables := make([]string, len(a.Templating.Variables))
				for i, t := range a.Templating.Variables {
					sub, err := run.EvaluateTemplate(t)
					if err != nil {
						logEvent(events.NewError(err))
					}
					templateVariables[i] = sub
				}

				evaluatedText = translation.Substitute(templateVariables)
				templating = flows.NewMsgTemplating(a.Templating.Template, translation.Language(), templateVariables)
			}
		}

		msg := flows.NewMsgOut(dest.URN.URN(), channelRef, evaluatedText, evaluatedAttachments, evaluatedQuickReplies, templating)
		logEvent(events.NewMsgCreated(msg))
	}

	// if we couldn't find a destination, create a msg without a URN or channel and it's up to the caller
	// to handle that as they want
	if len(destinations) == 0 {
		msg := flows.NewMsgOut(urns.NilURN, nil, evaluatedText, evaluatedAttachments, evaluatedQuickReplies, nil)
		logEvent(events.NewMsgCreated(msg))
	}

	return nil
}
