package actions

import (
	"github.com/nyaruka/gocommon/i18n"
	"github.com/nyaruka/gocommon/urns"
	"github.com/nyaruka/gocommon/uuids"
	"github.com/nyaruka/goflow/assets"
	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/flows/events"
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
//	{
//	  "uuid": "8eebd020-1af5-431c-b943-aa670fc74da9",
//	  "type": "send_msg",
//	  "text": "Hi @contact.name, are you ready to complete today's survey?",
//	  "attachments": [],
//	  "all_urns": false,
//	  "template": {
//	    "uuid": "3ce100b7-a734-4b4e-891b-350b1279ade2",
//	    "name": "revive_issue"
//	  },
//	  "template_variables": ["@contact.name"],
//	  "topic": "event"
//	}
//
// @action send_msg
type SendMsgAction struct {
	baseAction
	universalAction
	createMsgAction

	AllURNs           bool                      `json:"all_urns,omitempty"`
	Template          *assets.TemplateReference `json:"template,omitempty"`
	TemplateVariables []string                  `json:"template_variables,omitempty" engine:"localized,evaluated"`
	Topic             flows.MsgTopic            `json:"topic,omitempty" validate:"omitempty,msg_topic"`
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
func (a *SendMsgAction) Execute(run flows.Run, step flows.Step, logModifier flows.ModifierCallback, logEvent flows.EventCallback) error {
	// a message to a non-active contact is unsendable but can still be created
	unsendableReason := flows.NilUnsendableReason
	if run.Contact().Status() != flows.ContactStatusActive {
		unsendableReason = flows.UnsendableReasonContactStatus
	}

	content, lang := a.evaluateMessage(run, nil, a.Text, a.Attachments, a.QuickReplies, logEvent)
	locale := currentLocale(run, lang)

	destinations := run.Contact().ResolveDestinations(a.AllURNs)

	sa := run.Session().Assets()

	var template *flows.Template
	if a.Template != nil {
		template = sa.Templates().Get(a.Template.UUID)
	}

	// create a new message for each URN+channel destination
	for _, dest := range destinations {
		urn := dest.URN.URN()
		channelRef := assets.NewChannelReference(dest.Channel.UUID(), dest.Channel.Name())
		var msg *flows.MsgOut

		if template != nil {
			locales := []i18n.Locale{run.Session().MergedEnvironment().DefaultLocale(), run.Session().Environment().DefaultLocale()}
			translation := template.FindTranslation(dest.Channel, locales)
			if translation != nil {
				// TODO in future we won't be localizing template variables
				localizedVariables, _ := run.GetTextArray(uuids.UUID(a.UUID()), "template_variables", a.TemplateVariables, nil)

				// evaluate the variables
				evaluatedVariables := make([]string, len(localizedVariables))
				for i, varExp := range localizedVariables {
					v, _ := run.EvaluateTemplate(varExp, logEvent)
					evaluatedVariables[i] = v
				}

				templating := template.Templating(translation, evaluatedVariables)

				// the message we return is an approximate preview of what the channel will send using the template
				preview := translation.Preview(templating.Variables)
				locale := translation.Locale()

				msg = flows.NewMsgOut(urn, channelRef, preview, templating, flows.NilMsgTopic, locale, unsendableReason)
			}
		}

		if msg == nil {
			msg = flows.NewMsgOut(urn, channelRef, content, nil, a.Topic, locale, unsendableReason)
		}

		logEvent(events.NewMsgCreated(msg))
	}

	// if we couldn't find a destination, create a msg without a URN or channel and it's up to the caller
	// to handle that as they want
	if len(destinations) == 0 {
		msg := flows.NewMsgOut(urns.NilURN, nil, content, nil, a.Topic, locale, flows.UnsendableReasonNoDestination)
		logEvent(events.NewMsgCreated(msg))
	}

	return nil
}
