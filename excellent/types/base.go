package types

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/nyaruka/goflow/utils"

	"github.com/shopspring/decimal"
)

func init() {
	decimal.MarshalJSONWithoutQuotes = true
}

// XValue is the base interface of all Excellent types
type XValue interface {
	ToJSON() XString
	Reduce() XPrimitive
}

// XPrimitive is the base interface of all Excellent primitive types
type XPrimitive interface {
	XValue

	ToString() XString
	ToBool() XBool
}

// XResolvable is the interface for types which can be keyed into, e.g. foo.bar
type XResolvable interface {
	Resolve(key string) XValue
}

// XLengthable is the interface for types which have a length
type XLengthable interface {
	Length() int
}

// XIndexable is the interface for types which can be indexed into, e.g. foo.0. Such objects
// also need to be lengthable so that the engine knows what is a valid index and what isn't.
type XIndexable interface {
	XLengthable

	Index(index int) XValue
}

// XString is a string of characters
type XString string

// NewXString creates a new XString
func NewXString(value string) XString {
	return XString(value)
}

// Reduce returns the primitive version of this type (i.e. itself)
func (x XString) Reduce() XPrimitive { return x }

// ToString converts this type to a string
func (x XString) ToString() XString { return x }

// ToBool converts this type to a bool
func (x XString) ToBool() XBool { return string(x) != "" && strings.ToLower(string(x)) != "false" }

// ToJSON converts this type to JSON
func (x XString) ToJSON() XString { return RequireMarshalToXString(x.Native()) }

// Native returns the native value of this type
func (x XString) Native() string { return string(x) }

func (x XString) Length() int { return len(x) }

var XStringEmpty = NewXString("")
var _ XPrimitive = XStringEmpty
var _ XLengthable = XStringEmpty

// XNumber is any whole or fractional number
type XNumber decimal.Decimal

// NewXNumber creates a new XNumber
func NewXNumber(value decimal.Decimal) XNumber {
	return XNumber(value)
}

// NewXNumberFromInt creates a new XNumber from the given int
func NewXNumberFromInt(value int) XNumber {
	return XNumber(decimal.New(int64(value), 0))
}

// RequireXNumberFromString creates a new XNumber from the given string
func RequireXNumberFromString(value string) XNumber {
	return XNumber(decimal.RequireFromString(value))
}

// Reduce returns the primitive version of this type (i.e. itself)
func (x XNumber) Reduce() XPrimitive { return x }

// ToString converts this type to a string
func (x XNumber) ToString() XString { return XString(x.Native().String()) }

// ToBool converts this type to a bool
func (x XNumber) ToBool() XBool { return XBool(!x.Native().Equals(decimal.Zero)) }

// ToJSON converts this type to JSON
func (x XNumber) ToJSON() XString { return RequireMarshalToXString(x.Native()) }

// Native returns the native value of this type
func (x XNumber) Native() decimal.Decimal { return decimal.Decimal(x) }

var XNumberZero = XNumber(decimal.Zero)
var _ XPrimitive = XNumberZero

// XBool is a boolean true or false
type XBool bool

// NewXBool creates a new XBool
func NewXBool(value bool) XBool {
	return XBool(value)
}

// Reduce returns the primitive version of this type (i.e. itself)
func (x XBool) Reduce() XPrimitive { return x }

// ToString converts this type to a string
func (x XBool) ToString() XString { return XString(strconv.FormatBool(x.Native())) }

// ToBool converts this type to a bool
func (x XBool) ToBool() XBool { return x }

// ToJSON converts this type to JSON
func (x XBool) ToJSON() XString { return RequireMarshalToXString(x.Native()) }

// Native returns the native value of this type
func (x XBool) Native() bool { return bool(x) }

var XBoolFalse = NewXBool(false)
var XBoolTrue = NewXBool(true)
var _ XPrimitive = XBoolFalse

// XTime is a point in time
type XTime time.Time

// NewXTime creates a new XTime
func NewXTime(value time.Time) XTime {
	return XTime(value)
}

// Reduce returns the primitive version of this type (i.e. itself)
func (x XTime) Reduce() XPrimitive { return x }

// ToString converts this type to a string
func (x XTime) ToString() XString { return XString(utils.DateToISO(x.Native())) }

// ToBool converts this type to a bool
func (x XTime) ToBool() XBool { return XBool(!x.Native().IsZero()) }

// ToJSON converts this type to JSON
func (x XTime) ToJSON() XString { return RequireMarshalToXString(utils.DateToISO(x.Native())) }

// Native returns the native value of this type
func (x XTime) Native() time.Time { return time.Time(x) }

var XTimeZero = NewXTime(time.Time{})
var _ XPrimitive = XTimeZero

// XError is an error
type XError interface {
	XPrimitive
	error
}

type xerror struct {
	err error
}

// NewXError creates a new XError
func NewXError(err error) XError {
	return xerror{err: err}
}

// NewXResolveError creates a new XError when a key can't be resolved on an XResolvable
func NewXResolveError(resolvable XResolvable, key string) XError {
	return NewXError(fmt.Errorf("unable to resolve '%s' on %s", key, reflect.TypeOf(resolvable)))
}

// Reduce returns the primitive version of this type (i.e. itself)
func (x xerror) Reduce() XPrimitive { return x }

// ToString converts this type to a string
func (x xerror) ToString() XString { return XString(x.Native().Error()) }

// ToBool converts this type to a bool
func (x xerror) ToBool() XBool { return XBool(false) }

// ToJSON converts this type to JSON
func (x xerror) ToJSON() XString { return RequireMarshalToXString(x.Native().Error()) }

// Native returns the native value of this type
func (x xerror) Native() error { return x.err }

func (x xerror) Error() string { return x.err.Error() }

var NilXError = NewXError(nil)
var _ XError = NilXError

// RequireMarshalToXString calls json.Marshal in the given value and panics in the case of an error
func RequireMarshalToXString(x interface{}) XString {
	j, err := json.Marshal(x)
	if err != nil {
		panic(fmt.Sprintf("unable to marshal %v to JSON", x))
	}
	return XString(j)
}
