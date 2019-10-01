package flows

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/greatnonprofits-nfp/goflow/excellent/types"
	"github.com/greatnonprofits-nfp/goflow/utils"
)

// Result describes a value captured during a run's execution. It might have been implicitly created by a router, or explicitly
// created by a [set_run_result](#action:set_run_result) action. It renders as its value in a template, and has the following
// properties which can be accessed:
//
//  * `value` the value of the result
//  * `category` the category of the result
//  * `category_localized` the localized category of the result
//  * `input` the input associated with the result
//  * `node_uuid` the UUID of the node where the result was created
//  * `extra` any additional data associated with this result
//  * `created_on` the time when the result was created
//
// Examples:
//
//   @results -> 2Factor: 34634624463525\nFavorite Color: red\nPhone Number: +12344563452\nwebhook: 200
//   @results.favorite_color -> red
//   @results.favorite_color.value -> red
//   @results.favorite_color.category -> Red
//
// @context result
type Result struct {
	Name              string          `json:"name"`
	Value             string          `json:"value"`
	Category          string          `json:"category,omitempty"`
	CategoryLocalized string          `json:"category_localized,omitempty"`
	NodeUUID          NodeUUID        `json:"node_uuid"`
	Input             string          `json:"input,omitempty"`
	Extra             json.RawMessage `json:"extra,omitempty"`
	CreatedOn         time.Time       `json:"created_on"`
}

// NewResult creates a new result
func NewResult(name string, value string, category string, categoryLocalized string, nodeUUID NodeUUID, input string, extra json.RawMessage, createdOn time.Time) *Result {
	return &Result{
		Name:              name,
		Value:             value,
		Category:          category,
		CategoryLocalized: categoryLocalized,
		NodeUUID:          nodeUUID,
		Input:             input,
		Extra:             extra,
		CreatedOn:         createdOn,
	}
}

// Context returns the properties available in expressions
func (r *Result) Context(env utils.Environment) map[string]types.XValue {
	categoryLocalized := r.CategoryLocalized
	if categoryLocalized == "" {
		categoryLocalized = r.Category
	}

	return map[string]types.XValue{
		"__default__":          types.NewXText(r.Value),
		"name":                 types.NewXText(r.Name),
		"value":                types.NewXText(r.Value),
		"values":               types.NewXArray(types.NewXText(r.Value)),
		"category":             types.NewXText(r.Category),
		"categories":           types.NewXArray(types.NewXText(r.Category)),
		"category_localized":   types.NewXText(categoryLocalized),
		"categories_localized": types.NewXArray(types.NewXText(categoryLocalized)),
		"input":                types.NewXText(r.Input),
		"extra":                types.JSONToXValue(r.Extra),
		"node_uuid":            types.NewXText(string(r.NodeUUID)),
		"created_on":           types.NewXDateTime(r.CreatedOn),
	}
}

// Results is our wrapper around a map of snakified result names to result objects
type Results map[string]*Result

// NewResults creates a new empty set of results
func NewResults() Results {
	return make(Results, 0)
}

// Clone returns a clone of this results set
func (r Results) Clone() Results {
	clone := make(Results, len(r))
	for k, v := range r {
		clone[k] = v
	}
	return clone
}

// Save saves a new result in our map. The key is saved in a snakified format
func (r Results) Save(result *Result) {
	r[utils.Snakify(result.Name)] = result
}

// Get returns the result with the given key
func (r Results) Get(key string) *Result {
	return r[key]
}

// Context returns the properties available in expressions
func (r Results) Context(env utils.Environment) map[string]types.XValue {
	entries := make(map[string]types.XValue, len(r)+1)
	entries["__default__"] = types.NewXText(r.format())

	for k, v := range r {
		entries[k] = Context(env, v)
	}
	return entries
}

func (r Results) format() string {
	lines := make([]string, 0, len(r))
	for _, v := range r {
		lines = append(lines, fmt.Sprintf("%s: %s", v.Name, v.Value))
	}

	sort.Strings(lines)
	return strings.Join(lines, "\n")
}
