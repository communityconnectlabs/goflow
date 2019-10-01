package flows

import (
	"testing"

	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/assets/static/types"
	"github.com/greatnonprofits-nfp/goflow/utils"
	
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
		tt := NewTemplateTranslation(types.NewTemplateTranslation(*channel, utils.Language("eng"), tc.Content, len(tc.Variables)))
		result := tt.Substitute(tc.Variables)
		assert.Equal(t, tc.Expected, result, "%d: unexpected template substitution", i)
	}
}

func TestTemplates(t *testing.T) {
	channel1 := assets.NewChannelReference("0bce5fd3-c215-45a0-bcb8-2386eb194175", "Test Channel")
	tt1 := types.NewTemplateTranslation(*channel1, utils.Language("eng"), "Hello {{1}}", 1)
	tt2 := types.NewTemplateTranslation(*channel1, utils.Language("spa"), "Hola {{1}}", 1)
	template := NewTemplate(types.NewTemplate("c520cbda-e118-440f-aaf6-c0485088384f", "greeting", []*types.TemplateTranslation{tt1, tt2}))

	tas := NewTemplateAssets([]assets.Template{template})

	tcs := []struct {
		UUID      assets.TemplateUUID
		Channel   *assets.ChannelReference
		Languages []utils.Language
		Variables []string
		Expected  string
	}{
		{"c520cbda-e118-440f-aaf6-c0485088384f", channel1, []utils.Language{"eng", "spa"}, []string{"Chef"}, "Hello Chef"},
		{"c520cbda-e118-440f-aaf6-c0485088384f", channel1, []utils.Language{"deu", "spa"}, []string{"Chef"}, "Hola Chef"},
		{"c520cbda-e118-440f-aaf6-c0485088384f", nil, []utils.Language{"deu", "spa"}, []string{"Chef"}, ""},
		{"c520cbda-e118-440f-aaf6-c0485088384f", channel1, []utils.Language{"deu"}, []string{"Chef"}, ""},
		{"8c5d4910-114a-4521-ba1d-bde8b024865a", channel1, []utils.Language{"eng", "spa"}, []string{"Chef"}, ""},
	}

	for _, tc := range tcs {
		tr := tas.FindTranslation(tc.UUID, tc.Channel, tc.Languages)
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
