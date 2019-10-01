package main_test

import (
	"strings"
	"testing"

	main "github.com/greatnonprofits-nfp/goflow/cmd/flowmigrate"
	"github.com/greatnonprofits-nfp/goflow/test"

	"github.com/stretchr/testify/require"
)

func TestMigrate(t *testing.T) {
	input := strings.NewReader(`{
		"metadata": {
			"uuid": "76f0a02f-3b75-4b86-9064-e9195e1b3a02",
			"name": "Empty",
			"revision": 1
		},
		"base_language": "eng",
		"flow_type": "F",
		"action_sets": [],
		"rule_sets": []
	}`)

	migrated, err := main.Migrate(input, false, "")
	require.NoError(t, err)

	test.AssertEqualJSON(t, []byte(`{
		"uuid": "76f0a02f-3b75-4b86-9064-e9195e1b3a02",
		"name": "Empty",
		"spec_version": "13.0.0",
		"language": "eng",
		"type": "messaging",
		"revision": 1,
		"expire_after_minutes": 0,
		"localization": {},
		"nodes": [],
		"_ui": {
			"nodes": {},
			"stickies": {}
		}
	}`), migrated, "Migrated flow mismatch")
}
