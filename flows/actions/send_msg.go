package actions

import (
	"github.com/nyaruka/gocommon/urns"
	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/events"
	"github.com/greatnonprofits-nfp/goflow/utils"
)

func init() {
	RegisterType(TypeSendMsg, func() flows.Action { return &SendMsgAction{} })
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
	BaseAction
	universalAction
	createMsgAction

	AllURNs    bool        `json:"all_urns,omitempty"`
	Templating *Templating `json:"templating,omitempty" validate:"omitempty,dive"`
}

// Templating represents the templating that should be used if possible
type Templating struct {
	Template  *assets.TemplateReference `json:"template" validate:"required"`
	Variables []string                  `json:"variables"`
}

// NewSendMsgAction creates a new send msg action
func NewSendMsgAction(uuid flows.ActionUUID, text string, attachments []string, quickReplies []string, allURNs bool) *SendMsgAction {
	return &SendMsgAction{
		BaseAction: NewBaseAction(TypeSendMsg, uuid),
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
		logEvent(events.NewErrorEventf("can't execute action in session without a contact"))
		return nil
	}

	println("GoFlow")
	println(a.Text)

	evaluatedText, evaluatedAttachments, evaluatedQuickReplies := a.evaluateMessage(run, nil, a.Text, a.Attachments, a.QuickReplies, logEvent)

	destinations := run.Contact().ResolveDestinations(a.AllURNs)

	sa := run.Session().Assets()

	// create a new message for each URN+channel destination
	for _, dest := range destinations {
		var channelRef *assets.ChannelReference
		if dest.Channel != nil {
			channelRef = assets.NewChannelReference(dest.Channel.UUID(), dest.Channel.Name())
		}

		var templating *flows.MsgTemplating

		// do we have a template defined?
		if a.Templating != nil {
			translation := sa.Templates().FindTranslation(a.Templating.Template.UUID, channelRef, []utils.Language{run.Contact().Language(), run.Environment().DefaultLanguage()})
			if translation != nil {
				// evaluate our variables
				templateVariables := make([]string, len(a.Templating.Variables))
				for i, t := range a.Templating.Variables {
					sub, err := run.EvaluateTemplate(t)
					if err != nil {
						logEvent(events.NewErrorEvent(err))
					}
					templateVariables[i] = sub
				}

				evaluatedText = translation.Substitute(templateVariables)
				templating = flows.NewMsgTemplating(a.Templating.Template, translation.Language(), templateVariables)
			}
		}

		msg := flows.NewMsgOut(dest.URN.URN(), channelRef, evaluatedText, evaluatedAttachments, evaluatedQuickReplies, templating)
		logEvent(events.NewMsgCreatedEvent(msg))
	}

	// if we couldn't find a destination, create a msg without a URN or channel and it's up to the caller
	// to handle that as they want
	if len(destinations) == 0 {
		msg := flows.NewMsgOut(urns.NilURN, nil, evaluatedText, evaluatedAttachments, evaluatedQuickReplies, nil)
		logEvent(events.NewMsgCreatedEvent(msg))
	}

	return nil
}

// Inspect inspects this object and any children
func (a *SendMsgAction) Inspect(inspect func(flows.Inspectable)) {
	inspect(a)
	if a.Templating != nil {
		flows.InspectReference(a.Templating.Template, inspect)
	}
}

// EnumerateTemplates enumerates all expressions on this object and its children
func (a *SendMsgAction) EnumerateTemplates(include flows.TemplateIncluder) {
	include.String(&a.Text)
	include.Slice(a.Attachments)
	include.Slice(a.QuickReplies)
	if a.Templating != nil {
		include.Slice(a.Templating.Variables)
	}

	include.Translations(a, "text")
	include.Translations(a, "attachments")
	include.Translations(a, "quick_replies")
}
