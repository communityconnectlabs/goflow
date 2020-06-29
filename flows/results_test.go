package flows_test

import (
	"testing"
	"time"

	"github.com/greatnonprofits-nfp/goflow/envs"
	"github.com/greatnonprofits-nfp/goflow/excellent/types"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/test"

	"github.com/stretchr/testify/assert"
)

func TestResults(t *testing.T) {
	env := envs.NewBuilder().Build()

	result := flows.NewResult("Beer", "skol!", "Skol", "", flows.NodeUUID("26493ebb-a254-4461-a28d-c7761784e276"), "", nil, time.Date(2019, 4, 5, 14, 16, 30, 123456, time.UTC), "")
	results := flows.NewResults()
	results.Save(result)

	assert.Equal(t, result, results.Get("beer"))
	assert.Nil(t, results.Get("xxx"))

	test.AssertXEqual(t, types.NewXObject(map[string]types.XValue{
		"__default__": types.NewXText("Beer: skol!"),
		"beer": types.NewXObject(map[string]types.XValue{
			"__default__":          types.NewXText("skol!"),
			"category":             types.NewXText("Skol"),
			"categories":           types.NewXArray(types.NewXText("Skol")),
			"category_localized":   types.NewXText("Skol"),
			"categories_localized": types.NewXArray(types.NewXText("Skol")),
			"created_on":           types.NewXDateTime(time.Date(2019, 4, 5, 14, 16, 30, 123456, time.UTC)),
			"extra":                nil,
			"input":                types.XTextEmpty,
			"name":                 types.NewXText("Beer"),
			"node_uuid":            types.NewXText("26493ebb-a254-4461-a28d-c7761784e276"),
			"value":                types.NewXText("skol!"),
			"values":               types.NewXArray(types.NewXText("skol!")),
		}),
	}), flows.Context(env, results))
}
