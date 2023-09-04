package flows

import (
	"testing"

	"github.com/nyaruka/gocommon/i18n"
	"github.com/nyaruka/goflow/assets"
	"github.com/nyaruka/goflow/assets/static"
	"github.com/stretchr/testify/assert"
)

func TestTemplateTranslation(t *testing.T) {
	tcs := []struct {
		Content   string
		Variables []string
		Expected  string
	}{
		{"Hi {{1}}, {{2}}", []string{"Chef"}, "Hi Chef, "},
		{"Good boy {{1}}! Who's the best {{1}}?", []string{"Chef"}, "Good boy Chef! Who's the best Chef?"},
		{"Orbit {{1}}! No, go around the {{2}}!", []string{"Chef", "sofa"}, "Orbit Chef! No, go around the sofa!"},
	}

	channel := assets.NewChannelReference("0bce5fd3-c215-45a0-bcb8-2386eb194175", "Test Channel")

	for i, tc := range tcs {
		tt := NewTemplateTranslation(static.NewTemplateTranslation(*channel, i18n.Locale("eng-US"), tc.Content, len(tc.Variables), "a6a8863e_7879_4487_ad24_5e2ea429027c"))
		result := tt.Substitute(tc.Variables)
		assert.Equal(t, tc.Expected, result, "%d: unexpected template substitution", i)
	}
}

func TestTemplates(t *testing.T) {
	channel1 := assets.NewChannelReference("0bce5fd3-c215-45a0-bcb8-2386eb194175", "Test Channel")
	tt1 := static.NewTemplateTranslation(*channel1, i18n.Locale("eng"), "Hello {{1}}", 1, "")
	tt2 := static.NewTemplateTranslation(*channel1, i18n.Locale("spa-EC"), "Que tal {{1}}", 1, "")
	tt3 := static.NewTemplateTranslation(*channel1, i18n.Locale("spa-ES"), "Hola {{1}}", 1, "")
	template := NewTemplate(static.NewTemplate("c520cbda-e118-440f-aaf6-c0485088384f", "greeting", []*static.TemplateTranslation{tt1, tt2, tt3}))

	tas := NewTemplateAssets([]assets.Template{template})

	tcs := []struct {
		UUID      assets.TemplateUUID
		Channel   *assets.ChannelReference
		Locales   []i18n.Locale
		Variables []string
		Expected  string
	}{
		{
			"c520cbda-e118-440f-aaf6-c0485088384f",
			channel1,
			[]i18n.Locale{"eng-US", "spa-CO"},
			[]string{"Chef"},
			"Hello Chef",
		},
		{
			"c520cbda-e118-440f-aaf6-c0485088384f",
			channel1,
			[]i18n.Locale{"eng", "spa-CO"},
			[]string{"Chef"},
			"Hello Chef",
		},
		{
			"c520cbda-e118-440f-aaf6-c0485088384f",
			channel1,
			[]i18n.Locale{"deu-DE", "spa-ES"},
			[]string{"Chef"},
			"Hola Chef",
		},
		{
			"c520cbda-e118-440f-aaf6-c0485088384f",
			nil,
			[]i18n.Locale{"deu-DE", "spa-ES"},
			[]string{"Chef"},
			"",
		},
		{
			"c520cbda-e118-440f-aaf6-c0485088384f",
			channel1,
			[]i18n.Locale{"deu-DE"},
			[]string{"Chef"},
			"",
		},
		{
			"8c5d4910-114a-4521-ba1d-bde8b024865a",
			channel1,
			[]i18n.Locale{"eng-US", "spa-ES"},
			[]string{"Chef"},
			"",
		},
	}

	for _, tc := range tcs {
		tr := tas.FindTranslation(tc.UUID, tc.Channel, tc.Locales)
		if tr == nil {
			assert.Equal(t, "", tc.Expected)
			continue
		}
		assert.NotNil(t, tr.Asset())

		assert.Equal(t, tc.Expected, tr.Substitute(tc.Variables))
	}

	template = tas.Get(assets.TemplateUUID("c520cbda-e118-440f-aaf6-c0485088384f"))
	assert.NotNil(t, template)
	assert.Equal(t, assets.NewTemplateReference("c520cbda-e118-440f-aaf6-c0485088384f", "greeting"), template.Reference())
	assert.Equal(t, (*assets.TemplateReference)(nil), (*Template)(nil).Reference())
}
