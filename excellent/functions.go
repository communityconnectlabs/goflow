package excellent

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"unicode/utf8"

	"math"

	humanize "github.com/dustin/go-humanize"
	"github.com/nyaruka/goflow/utils"
	"github.com/shopspring/decimal"
)

// XFunction defines the interface that Excellent functions must implement
type XFunction func(env utils.Environment, args ...interface{}) interface{}

// XFUNCTIONS is our map of functions available in Excellent which aren't tests
var XFUNCTIONS = map[string]XFunction{
	"and": And,
	"if":  If,
	"or":  Or,

	"array_length": ArrayLength,
	"default":      Default,

	"legacy_add": LegacyAdd,

	"round":      Round,
	"round_up":   RoundUp,
	"round_down": RoundDown,
	"int":        Int,
	"max":        Max,
	"min":        Min,
	"mean":       Mean,
	"mod":        Mod,
	"rand":       Rand,
	"abs":        Abs,

	"fixed":     Fixed,
	"read_code": ReadCode,

	"char":              Char,
	"code":              Code,
	"split":             Split,
	"join":              Join,
	"title":             Title,
	"word":              Word,
	"remove_first_word": RemoveFirstWord,
	"word_count":        WordCount,
	"word_slice":        WordSlice,
	"field":             Field,
	"clean":             Clean,
	"left":              Left,
	"lower":             Lower,
	"length":            Length,
	"right":             Right,
	"string_length":     Length,
	"repeat":            Repeat,
	"replace":           Replace,
	"upper":             Upper,
	"percent":           Percent,

	"date":            Date,
	"date_from_parts": DateFromParts,
	"date_diff":       DateDiff,
	"date_add":        DateAdd,
	"day":             Day,
	"weekday":         Weekday,
	"month":           Month,
	"year":            Year,
	"hour":            Hour,
	"minute":          Minute,
	"second":          Second,
	"tz":              TZ,
	"tz_offset":       TZOffset,
	"today":           Today,
	"now":             Now,
}

//------------------------------------------------------------------------------------------
// Legacy Functions
//------------------------------------------------------------------------------------------

// LegacyAdd simulates our old + operator, which operated differently based on whether
// one of the parameters was a date or not. If one is a date, then the other side is
// expected to be an integer with a number of days to add to the date, otherwise a normal
// decimal addition is attempted.
func LegacyAdd(env utils.Environment, args ...interface{}) interface{} {
	if len(args) != 2 {
		return fmt.Errorf("LEGACY_ADD requires exactly two arguments, got %d", len(args))
	}

	// try to parse dates and decimals
	date1, date1Err := utils.ToDate(env, args[0])
	date2, date2Err := utils.ToDate(env, args[1])

	dec1, dec1Err := utils.ToDecimal(env, args[0])
	dec2, dec2Err := utils.ToDecimal(env, args[1])

	// if they are both dates, that's an error
	if date1Err == nil && date2Err == nil {
		return fmt.Errorf("LEGACY_ADD cannot operate on two dates")
	}

	// date and int, do a day addition
	if date1Err == nil && dec2Err == nil {
		if dec2.IntPart() < math.MinInt32 || dec2.IntPart() > math.MaxInt32 {
			return fmt.Errorf("LEGACY_ADD cannot operate on integers greater than 32 bit")
		}
		return date1.AddDate(0, 0, int(dec2.IntPart()))
	}

	// int and date, do a day addition
	if date2Err == nil && dec1Err == nil {
		if dec1.IntPart() < math.MinInt32 || dec1.IntPart() > math.MaxInt32 {
			return fmt.Errorf("LEGACY_ADD cannot operate on integers greater than 32 bit")
		}
		return date2.AddDate(0, 0, int(dec1.IntPart()))
	}

	// one of these doesn't look like a valid decimal either, bail
	if dec1Err != nil {
		return dec1Err
	}

	if dec2Err != nil {
		return dec2Err
	}

	// normal decimal addition
	return dec1.Add(dec2)
}

//------------------------------------------------------------------------------------------
// Utility Functions
//------------------------------------------------------------------------------------------

// ArrayLength returns the number of items in the passed in array
//
// array_length will return an error if it is passed an item which is not an array.
//
//    @(array_length(SPLIT("1 2 3", " "))) -> 3
//    @(array_length("123")) -> ERROR
//
// @function array_length
func ArrayLength(env utils.Environment, args ...interface{}) interface{} {
	if len(args) != 1 {
		return fmt.Errorf("ARRAY_LENGTH takes exactly one argument, got %d", len(args))
	}

	len, err := utils.SliceLength(args[0])
	if err != nil {
		return err
	}

	return len
}

// Default takes two arguments, returning the first if not an error or nil, otherwise the second
//
//   @(default(undeclared.var, "default_value")) -> default_value
//   @(default("10", "20")) -> 10
//   @(default(date("invalid-date"), "today")) -> today
//
// @function default
func Default(env utils.Environment, args ...interface{}) interface{} {
	if len(args) != 2 {
		return fmt.Errorf("DEFAULT takes exactly two arguments, got %d", len(args))
	}

	// first argument is nil, return arg2
	if args[0] == nil {
		return args[1]
	}

	// test whether arg1 is an error
	_, isErr := args[0].(error)
	if isErr {
		return args[1]
	}

	return args[0]
}

//------------------------------------------------------------------------------------------
// Bool Functions
//------------------------------------------------------------------------------------------

// And returns whether all the passed in arguments are truthy
//
//   @(and(true)) -> true
//   @(and(true, false, true)) -> false
//
// @function and
func And(env utils.Environment, args ...interface{}) interface{} {
	if len(args) == 0 {
		return fmt.Errorf("AND requires at least one argument")
	}

	val, err := utils.ToBool(env, args[0])
	if err != nil {
		return err
	}
	for _, iArg := range args[1:] {
		iVal, err := utils.ToBool(env, iArg)
		if err != nil {
			return err
		}
		val = val && iVal
	}
	return val
}

// Or returns whether if any of the passed in arguments are truthy
//
//   @(or(true)) -> true
//   @(or(true, false, true)) -> true
//
// @function or
func Or(env utils.Environment, args ...interface{}) interface{} {
	if len(args) == 0 {
		return fmt.Errorf("OR requires at least one argument")
	}

	val, err := utils.ToBool(env, args[0])
	if err != nil {
		return err
	}

	for _, iArg := range args[1:] {
		iVal, err := utils.ToBool(env, iArg)
		if err != nil {
			return err
		}
		val = val || iVal
	}
	return val
}

// If evaluates the first argument, and if truthy returns the 2nd, if not returning the 3rd
//
// If the first argument is an error that error is returned
//
//   @(if(1 = 1, "foo", "bar")) -> "foo"
//   @(if("foo" > "bar", "foo", "bar")) -> ERROR
//
// @function if
func If(env utils.Environment, args ...interface{}) interface{} {
	if len(args) != 3 {
		return fmt.Errorf("IF requires exactly 3 arguments, got %d", len(args))
	}

	truthy, err := utils.ToBool(env, args[0])
	if err != nil {
		return err
	}

	if truthy {
		return args[1]
	}

	return args[2]
}

//------------------------------------------------------------------------------------------
// Decimal Functions
//------------------------------------------------------------------------------------------

// Abs returns the absolute value of a number
//
//   @(abs(-10)) -> 10
//   @(abs(10.5)) -> 10.5
//   @(abs("foo")) -> ERROR
//
// @function abs
func Abs(env utils.Environment, args ...interface{}) interface{} {
	dec, err := checkOneDecimalArg(env, "ABS", args)
	if err != nil {
		return err
	}
	return dec.Abs()
}

// Round rounds the passed in number to the corresponding number of places
//
//   @(round(12.141, 2)) -> 12.14
//   @(round("notnum", 2)) -> ERROR
//
// @function round
func Round(env utils.Environment, args ...interface{}) interface{} {
	dec, round, err := checkTwoDecimalArgs(env, "ROUND", args)
	if err != nil {
		return err
	}

	roundInt := round.IntPart()
	if roundInt < 0 {
		return fmt.Errorf("ROUND decimal places argument must be valid 32 bit integer")
	}

	return dec.Round(int32(roundInt))
}

// RoundUp rounds up to the nearest integer value, also good at fighting weeds
//
//   @(round_up(12.141)) -> 13
//   @(round_up(12)) -> 12
//   @(round_up("foo")) -> ERROR
//
// @function round_up
func RoundUp(env utils.Environment, args ...interface{}) interface{} {
	dec, err := checkOneDecimalArg(env, "ROUND_UP", args)
	if err != nil {
		return err
	}

	return dec.Ceil()
}

// RoundDown rounds down to the nearest integer value
//
//   @(round_down(12.141)) -> 12
//   @(round_down(12.9)) -> 12
//   @(round_down("foo")) -> ERROR
//
// @function round_down
func RoundDown(env utils.Environment, args ...interface{}) interface{} {
	dec, err := checkOneDecimalArg(env, "ROUND_DOWN", args)
	if err != nil {
		return err
	}

	return dec.Floor()
}

// Int takes the passed in value and returns the integer value (floored)
//
//   @(int(12.14)) -> 12
//   @(int(12.9)) -> 12
//   @(int("foo")) -> ERROR
//
// @function int
func Int(env utils.Environment, args ...interface{}) interface{} {
	dec, err := checkOneDecimalArg(env, "INT", args)
	if err != nil {
		return err
	}

	return dec.Floor()
}

// Max takes a list of arguments and returns the greatest of them
//
//   @(max(1, 2)) -> 2
//   @(max(1, -1, 10)) -> 10
//   @(max(1, 10, "foo")) -> ERROR
//
// @function max
func Max(env utils.Environment, args ...interface{}) interface{} {
	if len(args) == 0 {
		return fmt.Errorf("MAX takes at least one argument")
	}

	max, err := utils.ToDecimal(env, args[0])
	if err != nil {
		return err
	}

	for _, v := range args[1:] {
		val, err := utils.ToDecimal(env, v)
		if err != nil {
			return err
		}

		if val.Cmp(max) > 0 {
			max = val
		}
	}
	return max
}

// Min takes a list of arguments and returns the smallest of them
//
//   @(min(1, 2)) -> 1
//   @(min(2, 2, -10)) -> -10
//   @(min(1, 2, "foo")) -> ERROR
//
// @function min
func Min(env utils.Environment, args ...interface{}) interface{} {
	if len(args) == 0 {
		return fmt.Errorf("MIN takes at least one argument")
	}

	max, err := utils.ToDecimal(env, args[0])
	if err != nil {
		return err
	}

	for _, v := range args[1:] {
		val, err := utils.ToDecimal(env, v)
		if err != nil {
			return err
		}

		if val.Cmp(max) < 0 {
			max = val
		}
	}
	return max
}

// Mean takes a list of numbers and returns the arithmetic mean of them
//
//   @(mean(1, 2)) -> 1.5
//   @(mean(1, 2, 6)) -> 3
//   @(mean(1, "foo")) -> ERROR
//
// @function mean
func Mean(env utils.Environment, args ...interface{}) interface{} {
	if len(args) == 0 {
		return fmt.Errorf("Mean requires at least one argument, got 0")
	}

	sum := decimal.Zero

	for _, val := range args {
		dec, err := utils.ToDecimal(env, val)
		if err != nil {
			return err
		}
		sum = sum.Add(dec)
	}

	return sum.Div(decimal.NewFromFloat(float64(len(args))))
}

// Mod returns the remainder of the division of the two arguments
//
//   @(mod(5, 2)) -> 1
//   @(mod(4, 2)) -> 0
//   @(mod(5, "foo")) -> ERROR
//
// @function mod
func Mod(env utils.Environment, args ...interface{}) interface{} {
	arg1, arg2, err := checkTwoDecimalArgs(env, "MOD", args)
	if err != nil {
		return err
	}

	return arg1.Mod(arg2)
}

var randSource = rand.NewSource(time.Now().UnixNano())

// Rand returns either a single random decimal between 0-1 or a random integer between the two passed parameters (inclusive)
//
//  @(rand()) == 0.5152
//  @(rand(1, 5)) == 3
//
// @function rand
func Rand(env utils.Environment, args ...interface{}) interface{} {
	if len(args) != 0 && len(args) != 2 {
		return fmt.Errorf("RAND takes either no arguments or two arguments, got %d", len(args))
	}

	if len(args) == 0 {
		return decimal.NewFromFloat(rand.New(randSource).Float64())
	}

	min, err := utils.ToDecimal(env, args[0])
	if err != nil {
		return err
	}
	max, err := utils.ToDecimal(env, args[1])
	if err != nil {
		return err
	}

	// turn to integers
	min = min.Floor()
	max = min.Floor()

	spread := min.Sub(max).Abs()

	// we add one here as the golang rand does is not inclusive, 2 will always return 1
	// since our contract is inclusive of both ends we need one more
	add := rand.New(randSource).Int63n(spread.IntPart() + 1)

	if min.Cmp(max) <= 0 {
		return min.Add(decimal.NewFromFloat(float64(add)))
	}
	return max.Add(decimal.NewFromFloat(float64(add)))
}

// Fixed returns the number formatted with the passed in number of decimal places and optional commas
//
//   @(fixed(31337, 2, true)) -> "31,337.00"
//   @(fixed(31337, 0, false)) -> "31337"
//   @(fixed("foo", 2, false)) -> ERROR
//
// @function fixed
func Fixed(env utils.Environment, args ...interface{}) interface{} {
	if len(args) != 3 {
		return fmt.Errorf("FIXED takes exactly three arguments, got %d", len(args))
	}

	dec, err := utils.ToDecimal(env, args[0])
	if err != nil {
		return err
	}

	places, err := utils.ToInt(env, args[1])
	if err != nil {
		return err
	}
	if places < 0 || places > 9 {
		return fmt.Errorf("FIXED must take 0-9 number of places, got %d", args[1])
	}

	commas, err := utils.ToBool(env, args[2])
	if err != nil {
		return err
	}

	// build our format string
	formatStr := bytes.Buffer{}
	if commas {
		formatStr.WriteString("#,###.")
	} else {
		formatStr.WriteString("####.")
	}
	if places > 0 {
		for i := 0; i < places; i++ {
			formatStr.WriteString("#")
		}
	}
	f64, _ := dec.Float64()
	return humanize.FormatFloat(formatStr.String(), f64)
}

//------------------------------------------------------------------------------------------
// IVR Functions
//------------------------------------------------------------------------------------------

// ReadCode converts the passed in string into something that can be read by IVR systems
//
// ReadCode will split the numbers such as they are easier to understand. This includes
// splitting in 3s or 4s if appropriate.
//
//   @(read_code("1234")) -> "1 2 3 4"
//   @(read_code("abc")) -> "a b c"
//   @(read_code("abcdef")) -> "a b c , d e f"
//
// @function read_code
func ReadCode(env utils.Environment, args ...interface{}) interface{} {
	if len(args) != 1 {
		return fmt.Errorf("READ_CODE takes exactly one argument, got %d", len(args))
	}

	// convert to a string
	val, err := utils.ToString(env, args[0])
	if err != nil {
		return err
	}

	var output bytes.Buffer

	// remove any leading +
	val = strings.TrimLeft(val, "+")

	length := len(val)

	// groups of three
	if length%3 == 0 {
		// groups of 3
		for i := 0; i < length; i += 3 {
			if i > 0 {
				output.WriteString(" , ")
			}
			output.WriteString(strings.Join(strings.Split(val[i:i+3], ""), " "))
		}
		return output.String()
	}

	// groups of four
	if length%4 == 0 {
		for i := 0; i < length; i += 4 {
			if i > 0 {
				output.WriteString(" , ")
			}
			output.WriteString(strings.Join(strings.Split(val[i:i+4], ""), " "))
		}
		return output.String()
	}

	// default, just do one at a time
	for i, c := range val {
		if i > 0 {
			output.WriteString(" , ")
		}
		output.WriteRune(c)
	}

	return output.String()
}

//------------------------------------------------------------------------------------------
// String Functions
//------------------------------------------------------------------------------------------

// Code returns the numeric code for the first character in the passed in string, it is the inverse of char
//
//   @(code("a")) -> "97"
//   @(code("abc")) -> "97"
//   @(code("😀")) -> "128512"
//   @(code("")) -> "ERROR"
//   @(code("15")) -> "49"
//   @(code(15)) -> "49"
//
// @function code
func Code(env utils.Environment, args ...interface{}) interface{} {
	str, err := checkOneStringArg(env, "code", args)
	if err != nil {
		return err
	}

	if len(str) == 0 {
		return fmt.Errorf("CODE requires a string of at least one character")
	}

	r, _ := utf8.DecodeRuneInString(str)
	return int(r)
}

// Split splits the passed in string based on the passed in delimeter
//
// Empty values are removed from the returned list
//
//   @(split("a b c", " ")) -> "a, b, c"
//   @(split("a", " ")) -> "a"
//   @(split("abc..d", ".")) -> "abc, d"
//   @(split("a.b.c.", ".")) -> "a, b, c"
//   @(split("a && b && c", " && ")) -> "a, b, c"
//
// @function split
func Split(env utils.Environment, args ...interface{}) interface{} {
	if len(args) != 2 {
		return fmt.Errorf("SPLIT takes exactly two arguments: string and delimiter, got %d", len(args))
	}

	s, err := utils.ToString(env, args[0])
	if err != nil {
		return err
	}

	sep, err := utils.ToString(env, args[1])
	if err != nil {
		return err
	}

	allSplits := strings.Split(s, sep)
	splits := make([]string, 0, len(allSplits))
	for i := range allSplits {
		if allSplits[i] != "" {
			splits = append(splits, allSplits[i])
		}
	}
	return splits
}

// Join joins the passed in slice using the passed in parameter
//
//   @(join(split("a.b.c", "."), " ")) -> "a b c"
//
// @function join
func Join(env utils.Environment, args ...interface{}) interface{} {
	if len(args) != 2 {
		return fmt.Errorf("JOIN takes exactly two arguments: the array to join and delimiter, got %d", len(args))
	}

	s, err := utils.ToStringArray(env, args[0])
	if err != nil {
		return err
	}

	sep, err := utils.ToString(env, args[1])
	if err != nil {
		return err
	}

	return strings.Join(s, sep)
}

// Char returns the rune for the passed in codepoint, which may be unicode, this is the reverse of code
//
//   @(char(33)) -> "!"
//   @(char(128512)) -> "😀"
//   @(char("foo")) -> ERROR
//
// @function char
func Char(env utils.Environment, args ...interface{}) interface{} {
	arg, err := checkOneDecimalArg(env, "CHAR", args)
	if err != nil {
		return err
	}

	return string(rune(arg.IntPart()))
}

// Title titlecases the passed in string, capitalizing each word
//
//   @(title("foo")) -> "Foo"
//   @(title("ryan lewis")) -> "Ryan Lewis"
//   @(title(123)) -> "123"
//
// @function title
func Title(env utils.Environment, args ...interface{}) interface{} {
	arg, err := checkOneStringArg(env, "TITLE", args)
	if err != nil {
		return err
	}

	return strings.Title(arg)
}

// Word returns the nth word in the passed in string
//
//   @(word("foo bar", 1)) -> "foo"
//   @(word("foo.bar", 1)) -> "foo"
//   @(word("one two.three", 3)) -> "three"
//
// @function word
func Word(env utils.Environment, args ...interface{}) interface{} {
	if len(args) != 2 {
		return fmt.Errorf("WORD takes exactly two arguments, got %d", len(args))
	}

	val, err := utils.ToString(env, args[0])
	if err != nil {
		return err
	}

	word, err := utils.ToInt(env, args[1])
	if err != nil {
		return err
	}

	words := utils.TokenizeString(val)
	if word-1 >= len(words) {
		return fmt.Errorf("Word offset %d is greater than number of words %d", word, len(words))
	}

	return words[word-1]
}

// RemoveFirstWord removes the 1st word of the passed in string
//
//   @(remove_first_word("foo bar")) -> "bar"
//
// @function remove_first_word
func RemoveFirstWord(env utils.Environment, args ...interface{}) interface{} {
	arg, err := checkOneStringArg(env, "REMOVE_FIRST_WORD", args)
	if err != nil {
		return err
	}

	words := utils.TokenizeString(arg)
	if len(words) > 1 {
		return strings.Join(words[1:], " ")
	}

	return ""
}

// WordSlice extracts a substring spanning from start up to but not-including stop, starting with 1
//
//   @(word_slice("foo bar", 1, 1)) -> "foo"
//   @(word_slice("foo bar", 1, 3)) -> "foo bar"
//   @(word_slice("foo bar", 3, 4)) -> ""
//
// @function word_slice
func WordSlice(env utils.Environment, args ...interface{}) interface{} {
	if len(args) != 3 {
		return fmt.Errorf("WORD_SLICE takes exactly three arguments, got %d", len(args))
	}

	arg, err := utils.ToString(env, args[0])
	if err != nil {
		return fmt.Errorf("WORD_SLICE requires a string as its first argument")
	}

	start, err := utils.ToInt(env, args[1])
	if err != nil || start <= 0 {
		return fmt.Errorf("WORD_SLICE must start with a postive index")
	}
	start--

	stop, err := utils.ToInt(env, args[2])
	if err != nil || start < 0 {
		return fmt.Errorf("WORD_SLICE must have a stop of 0 or greater")
	}

	words := utils.TokenizeString(arg)
	if start >= len(words) {
		return ""
	}

	if stop >= len(words) {
		stop = len(words)
	}

	if stop > 0 {
		return strings.Join(words[start:stop], " ")
	}
	return strings.Join(words[start:], " ")
}

// WordCount returns the number of words in the passed string
//
//   @(word_count("foo bar")) -> 2
//   @(word_count(10)) -> 1
//   @(word_count("")) -> 0
//   @(word_count("😀😃😄😁")) -> 4
//
// @function word_count
func WordCount(env utils.Environment, args ...interface{}) interface{} {
	arg, err := checkOneStringArg(env, "WORD_COUNT", args)
	if err != nil {
		return err
	}

	words := utils.TokenizeString(arg)
	return decimal.NewFromFloat(float64(len(words)))
}

// Field splits the string based on the passed in parameter and returns the nth field in that string. (first field is 1)
//
//   @(field("a,b,c", 2, ",")) -> "b"
//   @(field("a,b,c", 5, ",")) -> ""
//   @(field("a,b,c", "foo", ",")) -> ERROR
//
// @function field
func Field(env utils.Environment, args ...interface{}) interface{} {
	source, err := utils.ToString(env, args[0])
	if err != nil {
		return err
	}

	field, err := utils.ToInt(env, args[1])
	if err != nil {
		return err
	}

	if field < 0 {
		return fmt.Errorf("Cannot use a negative index to FIELD")
	}

	sep, err := utils.ToString(env, args[2])
	if err != nil {
		return err
	}

	fields := strings.Split(source, sep)
	if field-1 >= len(fields) {
		return ""
	}
	return strings.TrimSpace(fields[field-1])
}

// Clean strips any leading or trailing whitespace from the passed in string
//
//   @(clean("\nfoo\t")) -> "foo"
//   @(clean(" bar")) -> "bar"
//   @(clean(123)) -> "123"
//
// @function clean
func Clean(env utils.Environment, args ...interface{}) interface{} {
	arg, err := checkOneStringArg(env, "CLEAN", args)
	if err != nil {
		return err
	}

	return strings.TrimSpace(arg)
}

// Left returns the n most left characters of the passed in string
//
//   @(left("hello", 2)) -> "he"
//   @(left("hello", 7)) -> "hello"
//   @(left("😀😃😄😁", 2)) -> "😀😃"
//   @(left("hello", -1)) -> ERROR
//
// @function left
func Left(env utils.Environment, args ...interface{}) interface{} {
	str, l, err := checkOneStringOneIntArg(env, "LEFT", args)
	if err != nil {
		return err
	}

	// this weird construct does the right thing for multi-byte unicode
	var output bytes.Buffer
	i := 0
	for _, r := range str {
		if i >= l {
			break
		}
		output.WriteRune(r)
		i++
	}

	return output.String()
}

// Lower lowercases the passed in string
//
//   @(lower("HellO")) -> "hello"
//   @(lower("hello")) -> "hello"
//   @(lower("123")) -> "123"
//   @(lower("😀")) -> "😀"
//
// @function lower
func Lower(env utils.Environment, args ...interface{}) interface{} {
	arg, err := checkOneStringArg(env, "LOWER", args)
	if err != nil {
		return err
	}

	return strings.ToLower(arg)
}

// Right returns the n most right characters of the passed in string
//
//   @(right("hello", 2)) -> "lo"
//   @(right("hello", 7)) -> "hello"
//   @(right("😀😃😄😁", 2)) -> "😄😁"
//   @(right("hello", -1)) -> ERROR
//
// @function right
func Right(env utils.Environment, args ...interface{}) interface{} {
	str, l, err := checkOneStringOneIntArg(env, "RIGHT", args)
	if err != nil {
		return err
	}

	start := utf8.RuneCountInString(str) - l

	// this weird construct does the right thing for multi-byte unicode
	var output bytes.Buffer
	i := 0
	for _, r := range str {
		if i >= start {
			output.WriteRune(r)
		}
		i++
	}

	return output.String()
}

// Length returns the number of unicode characters in a string
//
//   @(length("Hello")) -> 5
//   @(length("😀😃😄😁")) -> 4
//   @(length(1234)) -> 4
//
// @function length
func Length(env utils.Environment, args ...interface{}) interface{} {
	arg, err := checkOneStringArg(env, "LENGTH", args)
	if err != nil {
		return err
	}

	return utf8.RuneCountInString(arg)
}

// Repeat return the first parameter repeated the second parameter number of times
//
//   @(repeat("*", 8)) -> "********"
//   @(repeat("*", "foo")) -> ERROR
//
// @function repeat
func Repeat(env utils.Environment, args ...interface{}) interface{} {
	str, i, err := checkOneStringOneIntArg(env, "REPEAT", args)
	if err != nil {
		return err
	}

	if i < 0 {
		return fmt.Errorf("REPEAT must be called with a positive integer, got %d", i)
	}

	var output bytes.Buffer
	for j := 0; j < i; j++ {
		output.WriteString(str)
	}

	return output.String()
}

// Replace replaces all occurrences of the first argument with the second argument
//
//   @(replace("foo bar", "foo", "zap")) -> "zap bar"
//   @(replace("foo bar", "baz", "zap")) -> "foo bar"
//
// @function replace
func Replace(env utils.Environment, args ...interface{}) interface{} {
	if len(args) != 3 {
		return fmt.Errorf("REPLACE takes exactly three arguments, got %d", len(args))
	}

	source, err := utils.ToString(env, args[0])
	if err != nil {
		return err
	}

	find, err := utils.ToString(env, args[1])
	if err != nil {
		return err
	}

	replace, err := utils.ToString(env, args[2])
	if err != nil {
		return err
	}

	return strings.Replace(source, find, replace, -1)
}

// Upper uppercases all characters in the passed in string
//
//   @(upper("Asdf")) -> "ASDF"
//   @(upper(123)) -> "123"
//
// @function upper
func Upper(env utils.Environment, args ...interface{}) interface{} {
	str, err := checkOneStringArg(env, "UPPER", args)
	if err != nil {
		return err
	}
	return strings.ToUpper(str)
}

// Percent converts the passed in decimal value to a string represented as a percentage
//
//   @(percent(0.54234)) -> "54%"
//   @(percent(1.2)) -> "120%"
//   @(percent("foo")) -> ERROR
//
// @function percent
func Percent(env utils.Environment, args ...interface{}) interface{} {
	dec, err := checkOneDecimalArg(env, "PERCENT", args)
	if err != nil {
		return err
	}

	// multiply by 100 and floor
	percent := dec.Mul(decimal.NewFromFloat(100)).Round(0)

	// add on a %
	return fmt.Sprintf("%d%%", percent.IntPart())
}

//------------------------------------------------------------------------------------------
// Date & Time Functions
//------------------------------------------------------------------------------------------

// Date turns the passed in string into a date according to the environment's settings
//
// date will return an error if it is unable to convert the string to a date.
//
//   @(date("1979-07-18")) -> 1979-07-18 00:00
//   @(date("2010 05 10")) -> 2010-05-10 00:00
//   @(date("NOT DATE")) -> ERROR
//
// @function date
func Date(env utils.Environment, args ...interface{}) interface{} {
	if len(args) != 1 {
		return fmt.Errorf("DATE requires exactly one argument, got %d", len(args))
	}
	arg1, err := utils.ToString(env, args[0])
	if err != nil {
		return err
	}

	date, err := utils.DateFromString(env, arg1)
	if err != nil {
		return err
	}

	return date
}

// DateFromParts converts the passed in year, month and day
//
//   @(date_from_parts(2017, 1, 15)) -> "2017-01-15 00:00"
//   @(date_from_parts(2017, 2, 31)) -> "2017-03-03 00:00"
//   @(date_from_parts(2017, 13, 15)) -> ERROR
//
// @function date_from_parts
func DateFromParts(env utils.Environment, args ...interface{}) interface{} {
	if len(args) != 3 {
		return fmt.Errorf("DATE_FROM_PARTS requires three arguments, got %d", len(args))
	}
	year, err := utils.ToInt(env, args[0])
	if err != nil {
		return err
	}
	month, err := utils.ToInt(env, args[1])
	if err != nil {
		return err
	}
	if month < 1 || month > 12 {
		return fmt.Errorf("Invalidate value for month, must be 1-12")
	}

	day, err := utils.ToInt(env, args[2])
	if err != nil {
		return err
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, env.Timezone())
}

// DateDiff returns the duration between two dates as an integer.
//
// Valid durations are "Y" for years, "M" for months, "W" for weeks, "D" for days, h" for hour,
// "m" for minutes, "s" for seconds
//
//   @(date_diff("2017-01-17", "2017-01-15", "D")) -> 2
//   @(date_diff("2017-01-17 10:50", "2017-01-17 12:30", "h")) -> -1
//   @(date_diff("2017-01-17", "2015-12-17", "Y")) -> 2
//
// @function date_diff
func DateDiff(env utils.Environment, args ...interface{}) interface{} {
	if len(args) != 3 {
		return fmt.Errorf("DATE_DIFF takes exactly three arguments, received %d", len(args))
	}

	date1, err := utils.ToDate(env, args[0])
	if err != nil {
		return err
	}

	date2, err := utils.ToDate(env, args[1])
	if err != nil {
		return err
	}

	unit, err := utils.ToString(env, args[2])
	if err != nil {
		return err
	}

	// find the duration between our dates
	duration := date1.Sub(date2)

	// then convert based on our unit
	switch unit {

	case "s":
		return int(duration / time.Second)

	case "m":
		return int(duration / time.Minute)

	case "h":
		return int(duration / time.Hour)

	case "D":
		return utils.DaysBetween(date1, date2)

	case "W":
		return int(utils.DaysBetween(date1, date2) / 7)

	case "M":
		return utils.MonthsBetween(date1, date2)

	case "Y":
		return date1.Year() - date2.Year()
	}

	return fmt.Errorf("Unknown unit: %s, must be one of s, m, h, D, W, M, Y", unit)
}

// DateAdd calculates the date value arrived at by adding the number of units to the passed in date
//
// Valid durations are "Y" for years, "M" for months, "W" for weeks, "D" for days, h" for hour,
// "m" for minutes, "s" for seconds
//
//   @(date_add("2017-01-15", 5, "D")) -> "2017-01-20 00:00"
//   @(date_add("2017-01-15 10:45", 30, "m")) -> "2017-01-15 11:15"
//
// @function date_add
func DateAdd(env utils.Environment, args ...interface{}) interface{} {
	if len(args) != 3 {
		return fmt.Errorf("DATE_ADD takes exactly three arguments, received %d", len(args))
	}

	date, err := utils.ToDate(env, args[0])
	if err != nil {
		return err
	}

	duration, err := utils.ToInt(env, args[1])
	if err != nil {
		return err
	}

	unit, err := utils.ToString(env, args[2])
	if err != nil {
		return err
	}

	switch unit {

	case "s":
		return date.Add(time.Duration(duration) * time.Second)

	case "m":
		return date.Add(time.Duration(duration) * time.Minute)

	case "h":
		return date.Add(time.Duration(duration) * time.Hour)

	case "D":
		return date.AddDate(0, 0, duration)

	case "W":
		return date.AddDate(0, 0, duration*7)

	case "M":
		return date.AddDate(0, duration, 0)

	case "Y":
		return date.AddDate(duration, 0, 0)
	}

	return fmt.Errorf("Unknown unit: %s, must be one of s, m, h, d, w, M, y", unit)
}

// Day returns the day of the month for the passed in date
//
//   @(day("2017-01-15")) -> 15
//   @(day("foo")) -> ERROR
//
// @function day
func Day(env utils.Environment, args ...interface{}) interface{} {
	date, err := checkOneDateArg(env, "DAY", args)
	if err != nil {
		return err
	}

	return date.Day()
}

// Weekday returns the day of the week for the passed in date, 0 is sunday, 1 is monday..
//
//   @(weekday("2017-01-15")) -> 0
//   @(weekday("foo")) -> ERROR
//
// @function weekday
func Weekday(env utils.Environment, args ...interface{}) interface{} {
	date, err := checkOneDateArg(env, "WEEKDAY", args)
	if err != nil {
		return err
	}

	return int(date.Weekday())
}

// Month returns the month of the year for the passed in date
//
//   @(month("2017-01-15")) -> 1
//   @(month("foo")) -> ERROR
//
// @function month
func Month(env utils.Environment, args ...interface{}) interface{} {
	date, err := checkOneDateArg(env, "MONTH", args)
	if err != nil {
		return err
	}

	return int(date.Month())
}

// Year returns the year for the passed in date
//
//   @(year("2017-01-15")) -> 2017
//   @(year("foo")) -> ERROR
//
// @function year
func Year(env utils.Environment, args ...interface{}) interface{} {
	date, err := checkOneDateArg(env, "YEAR", args)
	if err != nil {
		return err
	}

	return int(date.Year())
}

// Hour returns the hour of the day (0-24) for the passed in date
//
//   @(hour("2017-01-15 02:15:18PM")) -> 14
//   @(hour("2017-01-15 00:15:00AM")) -> 0
//   @(hour("2017-01-15")) -> 0
//   @(hour("foo")) -> ERROR
//
// @function hour
func Hour(env utils.Environment, args ...interface{}) interface{} {
	date, err := checkOneDateArg(env, "HOUR", args)
	if err != nil {
		return err
	}

	return int(date.Hour())
}

// Minute returns the minute of the hour for the passed in date
//
//   @(minute("2017-01-15 02:15:18PM")) -> 15
//   @(minute("2017-01-15")) -> 0
//   @(minute("foo")) -> ERROR
//
// @function minute
func Minute(env utils.Environment, args ...interface{}) interface{} {
	date, err := checkOneDateArg(env, "MINUTE", args)
	if err != nil {
		return err
	}

	return int(date.Minute())
}

// Second returns the second of the minute for the passed in date
//
//   @(second("2017-01-15 02:15:18PM")) -> 18
//   @(second("2017-01-15 02:15")) -> 0
//   @(second("2017-01-15")) -> 0
//   @(second("foo")) -> ERROR
//
// @function second
func Second(env utils.Environment, args ...interface{}) interface{} {
	date, err := checkOneDateArg(env, "SECOND", args)
	if err != nil {
		return err
	}

	return int(date.Second())
}

// TZ returns the timezone for the passed in date
//
// If not timezone information is present in the date, then the environment's
// timezone will be returned
//
//   @(tz("2017-01-15 02:15:18PM UTC")) -> "UTC"
//   @(tz("2017-01-15 02:15:18PM")) -> "UTC"
//   @(tz("2017-01-15")) -> "UTC"
//   @(tz("foo")) -> ERROR
//
// @function tz
func TZ(env utils.Environment, args ...interface{}) interface{} {
	date, err := checkOneDateArg(env, "TZ", args)
	if err != nil {
		return err
	}

	return date.Location().String()
}

// TZOffset returns the offset for the timezone as a string +/- HHMM
//
// If no timezone information is present in the date, then the environment's
// timezone offset will be returned
//
//   @(tz_offset("2017-01-15 02:15:18PM UTC")) -> "+0000"
//   @(tz_offset("2017-01-15 02:15:18PM")) -> "+0000"
//   @(tz_offset("2017-01-15")) -> "+0000"
//   @(tz_offset("foo")) -> ERROR
//
// @function tz_offset
func TZOffset(env utils.Environment, args ...interface{}) interface{} {
	date, err := checkOneDateArg(env, "TZ_OFFSET", args)
	if err != nil {
		return err
	}

	// this looks like we are returning a set offset, but this is how go describes formats
	return date.Format("-0700")

}

// Today returns the current date in the current timezone, time is set to midnight in the environment timezone
//
//  @(today()) -> "2017-01-15 00:00"
//
// @function today
func Today(env utils.Environment, args ...interface{}) interface{} {
	if len(args) > 0 {
		return fmt.Errorf("TODAY takes no arguments, got %d", len(args))
	}

	nowTZ := time.Now().In(env.Timezone())
	return time.Date(nowTZ.Year(), nowTZ.Month(), nowTZ.Day(), 0, 0, 0, 0, env.Timezone())
}

// Now returns the current date and time in the environment timezone
//
//  @(now()) -> "2017-01-15 02:15"
//
// @function now
func Now(env utils.Environment, args ...interface{}) interface{} {
	if len(args) > 0 {
		return fmt.Errorf("NOW takes no arguments, got %d", len(args))
	}

	return time.Now().In(env.Timezone())
}

//----------------------------------------------------------------------------------------
// Utility Functions
//----------------------------------------------------------------------------------------

func checkOneDecimalArg(env utils.Environment, funcName string, args []interface{}) (decimal.Decimal, error) {
	if len(args) != 1 {
		return decimal.Zero, fmt.Errorf("%s takes exactly one argument, got %d", funcName, len(args))
	}

	arg1, err := utils.ToDecimal(env, args[0])
	if err != nil {
		return decimal.Zero, err
	}

	return arg1, nil
}

func checkOneStringArg(env utils.Environment, funcName string, args []interface{}) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("%s takes exactly one argument, got %d", funcName, len(args))
	}

	arg1, err := utils.ToString(env, args[0])
	if err != nil {
		return "", err
	}

	return arg1, nil
}

func checkOneStringOneIntArg(env utils.Environment, funcName string, args []interface{}) (string, int, error) {
	if len(args) != 2 {
		return "", 0, fmt.Errorf("%s takes exactly two arguments, got %d", funcName, len(args))
	}

	arg1, err := utils.ToString(env, args[0])
	if err != nil {
		return "", 0, err
	}

	arg2, err := utils.ToInt(env, args[1])
	if err != nil {
		return "", 0, err
	}

	return arg1, arg2, err
}

func checkTwoDecimalArgs(env utils.Environment, funcName string, args []interface{}) (decimal.Decimal, decimal.Decimal, error) {
	if len(args) != 2 {
		return decimal.Zero, decimal.Zero, fmt.Errorf("%s takes exactly two arguments, got %d", funcName, len(args))
	}

	arg1, err := utils.ToDecimal(env, args[0])
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}

	arg2, err := utils.ToDecimal(env, args[1])
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}

	return arg1, arg2, nil
}

func checkOneDateArg(env utils.Environment, funcName string, args []interface{}) (time.Time, error) {
	if len(args) != 1 {
		return utils.ZeroTime, fmt.Errorf("%s takes exactly one argument, got %d", funcName, len(args))
	}

	arg1, err := utils.ToDate(env, args[0])
	if err != nil {
		return utils.ZeroTime, err
	}

	return arg1, err
}
