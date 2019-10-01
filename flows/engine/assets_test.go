package engine_test

import (
	"testing"

	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/assets/static"
	"github.com/greatnonprofits-nfp/goflow/flows/engine"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var assetsJSON = `{
	"groups": [
		{
			"uuid": "2aad21f6-30b7-42c5-bd7f-1b720c154817",
			"name": "Survey Audience"
		}
	]
}`

func TestSessionAssets(t *testing.T) {
	source, err := static.NewSource([]byte(assetsJSON))
	require.NoError(t, err)

	sessionAssets, err := engine.NewSessionAssets(source)
	require.NoError(t, err)

	group := sessionAssets.Groups().Get(assets.GroupUUID("2aad21f6-30b7-42c5-bd7f-1b720c154817"))
	assert.NotNil(t, group)
	assert.Equal(t, assets.GroupUUID("2aad21f6-30b7-42c5-bd7f-1b720c154817"), group.UUID())
	assert.Equal(t, "Survey Audience", group.Name())
}
