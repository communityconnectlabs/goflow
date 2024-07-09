package flows

import (
	"fmt"
	"slices"

	"github.com/go-playground/validator/v10"
	"github.com/nyaruka/gocommon/i18n"
	"github.com/nyaruka/gocommon/urns"
	"github.com/nyaruka/gocommon/uuids"
	"github.com/nyaruka/goflow/assets"
	"github.com/nyaruka/goflow/envs"
	"github.com/nyaruka/goflow/utils"
)

func init() {
	utils.RegisterValidatorAlias("msg_topic", "eq=event|eq=account|eq=purchase|eq=agent", func(validator.FieldError) string {
		return "is not a valid message topic"
	})
}

type UnsendableReason string

const (
	// max length of a message attachment (type:url)
	MaxAttachmentLength = 2048

	// max length of a quick reply
	MaxQuickReplyLength = 64

	NilUnsendableReason           UnsendableReason = ""
	UnsendableReasonNoDestination UnsendableReason = "no_destination" // no sendable channel+URN pair
	UnsendableReasonContactStatus UnsendableReason = "contact_status" // contact is blocked or stopped or archived
)

// MsgTopic is the topic, as required by some channel types
type MsgTopic string

// possible msg topic values
const (
	NilMsgTopic      MsgTopic = ""
	MsgTopicEvent    MsgTopic = "event"
	MsgTopicAccount  MsgTopic = "account"
	MsgTopicPurchase MsgTopic = "purchase"
	MsgTopicAgent    MsgTopic = "agent"
)

// BaseMsg represents a incoming or outgoing message with the session contact
type BaseMsg struct {
	UUID_        MsgUUID                  `json:"uuid"`
	ID_          MsgID                    `json:"id,omitempty"`
	URN_         urns.URN                 `json:"urn,omitempty" validate:"omitempty,urn"`
	Channel_     *assets.ChannelReference `json:"channel,omitempty"`
	Text_        string                   `json:"text"`
	Attachments_ []utils.Attachment       `json:"attachments,omitempty"`
}

// MsgIn represents a incoming message from the session contact
type MsgIn struct {
	BaseMsg

	ExternalID_ string `json:"external_id,omitempty"`
}

// MsgOut represents a outgoing message to the session contact
type MsgOut struct {
	BaseMsg

	QuickReplies_     []string         `json:"quick_replies,omitempty"`
	Templating_       *MsgTemplating   `json:"templating,omitempty"`
	Topic_            MsgTopic         `json:"topic,omitempty"`
	Locale_           i18n.Locale      `json:"locale,omitempty"`
	UnsendableReason_ UnsendableReason `json:"unsendable_reason,omitempty"`
}

// NewMsgIn creates a new incoming message
func NewMsgIn(uuid MsgUUID, urn urns.URN, channel *assets.ChannelReference, text string, attachments []utils.Attachment) *MsgIn {
	return &MsgIn{
		BaseMsg: BaseMsg{
			UUID_:        uuid,
			URN_:         urn,
			Channel_:     channel,
			Text_:        text,
			Attachments_: attachments,
		},
	}
}

// NewMsgOut creates a new outgoing message
func NewMsgOut(urn urns.URN, channel *assets.ChannelReference, content *MsgContent, templating *MsgTemplating, topic MsgTopic, locale i18n.Locale, reason UnsendableReason) *MsgOut {
	return &MsgOut{
		BaseMsg: BaseMsg{
			UUID_:        MsgUUID(uuids.New()),
			URN_:         urn,
			Channel_:     channel,
			Text_:        content.Text,
			Attachments_: content.Attachments,
		},
		QuickReplies_:     content.QuickReplies,
		Templating_:       templating,
		Topic_:            topic,
		Locale_:           locale,
		UnsendableReason_: reason,
	}
}

// NewIVRMsgOut creates a new outgoing message for IVR
func NewIVRMsgOut(urn urns.URN, channel *assets.ChannelReference, text string, audioURL string, locale i18n.Locale) *MsgOut {
	var attachments []utils.Attachment
	if audioURL != "" {
		attachments = []utils.Attachment{utils.Attachment(fmt.Sprintf("audio:%s", audioURL))}
	}

	return &MsgOut{
		BaseMsg: BaseMsg{
			UUID_:        MsgUUID(uuids.New()),
			URN_:         urn,
			Channel_:     channel,
			Text_:        text,
			Attachments_: attachments,
		},
		QuickReplies_: nil,
		Templating_:   nil,
		Topic_:        NilMsgTopic,
		Locale_:       locale,
	}
}

// UUID returns the UUID of this message
func (m *BaseMsg) UUID() MsgUUID { return m.UUID_ }

// ID returns the internal ID of this message
func (m *BaseMsg) ID() MsgID { return m.ID_ }

// SetID sets the internal ID of this message
func (m *BaseMsg) SetID(id MsgID) { m.ID_ = id }

// URN returns the URN of this message
func (m *BaseMsg) URN() urns.URN { return m.URN_ }

// SetURN returns the URN of this message
func (m *BaseMsg) SetURN(urn urns.URN) { m.URN_ = urn }

// Channel returns the channel of this message
func (m *BaseMsg) Channel() *assets.ChannelReference { return m.Channel_ }

// Text returns the text of this message
func (m *BaseMsg) Text() string { return m.Text_ }

// Attachments returns the attachments of this message
func (m *BaseMsg) Attachments() []utils.Attachment { return m.Attachments_ }

// ExternalID returns the optional external ID of this incoming message
func (m *MsgIn) ExternalID() string { return m.ExternalID_ }

// SetExternalID sets the external ID of this message
func (m *MsgIn) SetExternalID(id string) { m.ExternalID_ = id }

// QuickReplies returns the quick replies of this outgoing message
func (m *MsgOut) QuickReplies() []string { return m.QuickReplies_ }

// Templating returns the templating to use to send this message (if any)
func (m *MsgOut) Templating() *MsgTemplating { return m.Templating_ }

// Topic returns the topic to use to send this message (if any)
func (m *MsgOut) Topic() MsgTopic { return m.Topic_ }

// Locale returns the locale of this message (if any)
func (m *MsgOut) Locale() i18n.Locale { return m.Locale_ }

// UnsendableReason returns the reason this message can't be sent (if any)
func (m *MsgOut) UnsendableReason() UnsendableReason { return m.UnsendableReason_ }

type TemplatingVariable struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type TemplatingComponent struct {
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	Variables map[string]int `json:"variables"`
}

// MsgTemplating represents any substituted message template that should be applied when sending this message
type MsgTemplating struct {
	Template   *assets.TemplateReference `json:"template"`
	Components []*TemplatingComponent    `json:"components,omitempty"`
	Variables  []*TemplatingVariable     `json:"variables,omitempty"`
}

// NewMsgTemplating creates and returns a new msg template
func NewMsgTemplating(template *assets.TemplateReference, components []*TemplatingComponent, variables []*TemplatingVariable) *MsgTemplating {
	return &MsgTemplating{Template: template, Components: components, Variables: variables}
}

// MsgContent is message content in a particular language
type MsgContent struct {
	Text         string             `json:"text"`
	Attachments  []utils.Attachment `json:"attachments,omitempty"`
	QuickReplies []string           `json:"quick_replies,omitempty"`
}

func (c *MsgContent) Empty() bool {
	return c.Text == "" && len(c.Attachments) == 0 && len(c.QuickReplies) == 0
}

type BroadcastTranslations map[i18n.Language]*MsgContent

// ForContact is a utility to help callers get the message content for a contact
func (b BroadcastTranslations) ForContact(e envs.Environment, c *Contact, baseLanguage i18n.Language) (*MsgContent, i18n.Locale) {
	// get the set of languages to merge translations from
	languages := make([]i18n.Language, 0, 3)

	// highest priority is the contact language if it is valid
	if c.Language() != i18n.NilLanguage && slices.Contains(e.AllowedLanguages(), c.Language()) {
		languages = append(languages, c.Language())
	}

	// then the default workspace language, then the base language
	languages = append(languages, e.DefaultLanguage(), baseLanguage)

	content := &MsgContent{}
	language := i18n.NilLanguage
	country := e.DefaultCountry()
	if c.Country() != i18n.NilCountry {
		country = c.Country()
	}

	for _, lang := range languages {
		trans := b[lang]
		if trans != nil {
			if content.Text == "" && trans.Text != "" {
				content.Text = trans.Text
				language = lang
			}
			if len(content.Attachments) == 0 && len(trans.Attachments) > 0 {
				content.Attachments = trans.Attachments
			}
			if len(content.QuickReplies) == 0 && len(trans.QuickReplies) > 0 {
				content.QuickReplies = trans.QuickReplies
			}
		}
	}

	return content, i18n.NewLocale(language, country)
}
