package static

import (
	"testing"

	"github.com/nyaruka/gocommon/i18n"
	"github.com/nyaruka/gocommon/jsonx"
	"github.com/nyaruka/goflow/assets"
	"github.com/stretchr/testify/assert"
)

func TestTemplate(t *testing.T) {
	channel := assets.NewChannelReference("Test Channel", "ffffffff-9b24-92e1-ffff-ffffb207cdb4")

	tp1 := NewTemplateParam("text")
	assert.Equal(t, "text", tp1.Type())

	tc1 := NewTemplateComponent("Hello {{1}}", []*TemplateParam{&tp1})

	translation := NewTemplateTranslation(channel, i18n.Locale("eng-US"), "0162a7f4_dfe4_4c96_be07_854d5dba3b2b", map[string]*TemplateComponent{"body": tc1})
	assert.Equal(t, channel, translation.Channel())
	assert.Equal(t, i18n.Locale("eng-US"), translation.Locale())
	assert.Equal(t, "0162a7f4_dfe4_4c96_be07_854d5dba3b2b", translation.Namespace())
	assert.Equal(t, map[string]assets.TemplateComponent{"body": (assets.TemplateComponent)(tc1)}, translation.Components())

	template := NewTemplate(assets.TemplateUUID("8a9c1f73-5059-46a0-ba4a-6390979c01d3"), "hello", []*TemplateTranslation{translation})
	assert.Equal(t, assets.TemplateUUID("8a9c1f73-5059-46a0-ba4a-6390979c01d3"), template.UUID())
	assert.Equal(t, "hello", template.Name())
	assert.Equal(t, 1, len(template.Translations()))

	// test json and back
	asJSON, err := jsonx.Marshal(template)
	assert.NoError(t, err)

	copy := Template{}
	err = jsonx.Unmarshal(asJSON, &copy)
	assert.NoError(t, err)

	assert.Equal(t, copy.Name(), template.Name())
	assert.Equal(t, copy.UUID(), template.UUID())
	assert.Equal(t, copy.Translations()[0].Namespace(), template.Translations()[0].Namespace())
}
