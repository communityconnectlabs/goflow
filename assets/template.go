package assets

import (
	"fmt"

	"github.com/nyaruka/gocommon/i18n"
	"github.com/nyaruka/gocommon/uuids"
)

// TemplateUUID is the UUID of a template
type TemplateUUID uuids.UUID

// Template is a message template, currently only used by WhatsApp channels
//
//	{
//	  "name": "revive-issue",
//	  "uuid": "14782905-81a6-4910-bc9f-93ad287b23c3",
//	  "translations": [
//	    {
//	       "locale": "eng-US",
//	       "content": "Hi {{1}}, are you still experiencing your issue?",
//	       "channel": {
//	         "uuid": "cf26be4c-875f-4094-9e08-162c3c9dcb5b",
//	         "name": "Twilio Channel"
//	       }
//	    },
//	    {
//	       "locale": "fra",
//	       "content": "Bonjour {{1}}",
//	       "channel": {
//	         "uuid": "cf26be4c-875f-4094-9e08-162c3c9dcb5b",
//	         "name": "Twilio Channel"
//	       }
//	    }
//	  ]
//	}
//
// @asset template
type Template interface {
	UUID() TemplateUUID
	Name() string
	Translations() []TemplateTranslation
}

// TemplateParam is a parameter for template translation
type TemplateParam interface {
	Type() string
}

type TemplateComponent interface {
	Content() string
	Params() []TemplateParam
}

// TemplateTranslation represents a single translation for a specific template and channel
type TemplateTranslation interface {
	Locale() i18n.Locale
	Namespace() string
	Channel() *ChannelReference
	Components() map[string]TemplateComponent
}

// TemplateReference is used to reference a Template
type TemplateReference struct {
	UUID TemplateUUID `json:"uuid" validate:"required,uuid"`
	Name string       `json:"name"`
}

// NewTemplateReference creates a new template reference with the given UUID and name
func NewTemplateReference(uuid TemplateUUID, name string) *TemplateReference {
	return &TemplateReference{UUID: uuid, Name: name}
}

// GenericUUID returns the untyped UUID
func (r *TemplateReference) GenericUUID() uuids.UUID {
	return uuids.UUID(r.UUID)
}

// Identity returns the unique identity of the asset
func (r *TemplateReference) Identity() string {
	return string(r.UUID)
}

// Type returns the name of the asset type
func (r *TemplateReference) Type() string {
	return "template"
}

func (r *TemplateReference) String() string {
	return fmt.Sprintf("%s[uuid=%s,name=%s]", r.Type(), r.Identity(), r.Name)
}

// Variable returns whether this a variable (vs concrete) reference
func (r *TemplateReference) Variable() bool {
	return false
}

var _ UUIDReference = (*TemplateReference)(nil)
