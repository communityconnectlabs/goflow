package functions

import (
	"strings"

	"github.com/greatnonprofits-nfp/goflow/excellent/types"
	"github.com/greatnonprofits-nfp/goflow/utils"
)

// XFUNCTIONS is our map of functions available in Excellent which aren't tests
var XFUNCTIONS = map[string]types.XFunction{}

// RegisterXFunction registers a new function in Excellent
func RegisterXFunction(name string, function types.XFunction) {
	XFUNCTIONS[name] = function
}

// Lookup returns the function with the given name (case-insensitive) or nil
func Lookup(name string) types.XFunction {
	return XFUNCTIONS[strings.ToLower(name)]
}

// Call calls the given function with the given parameters
func Call(env utils.Environment, name string, function types.XFunction, params []types.XValue) types.XValue {
	val := function(env, params...)

	// if function returned an error, wrap the error with the function name
	if types.IsXError(val) {
		return types.NewXErrorf("error calling %s: %s", strings.ToUpper(name), val.(types.XError).Error())
	}

	return val
}
