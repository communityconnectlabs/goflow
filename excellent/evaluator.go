package excellent

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/nyaruka/goflow/excellent/gen"
	"github.com/nyaruka/goflow/excellent/types"
	"github.com/nyaruka/goflow/utils"

	"github.com/antlr/antlr4/runtime/Go/antlr"
)

// EvaluateExpression evalutes the passed in template, returning the typed value it evaluates to, which might be an error
func EvaluateExpression(env utils.Environment, context types.XValue, expression string) types.XValue {
	errListener := NewErrorListener(expression)

	input := antlr.NewInputStream(expression)
	lexer := gen.NewExcellent2Lexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	p := gen.NewExcellent2Parser(stream)
	p.RemoveErrorListeners()
	p.AddErrorListener(errListener)
	tree := p.Parse()

	// if we ran into errors parsing, return the first one
	if len(errListener.Errors()) > 0 {
		return errListener.Errors()[0]
	}

	visitor := NewVisitor(env, context)
	return toXValue(visitor.Visit(tree))
}

// EvaluateTemplate tries to evaluate the passed in template into an object, this only works if the template
// is a single identifier or expression, ie: "@contact" or "@(first(contact.urns))". In cases
// which are not a single identifier or expression, we return the stringified value
func EvaluateTemplate(env utils.Environment, context types.XValue, template string, allowedTopLevels []string) (types.XValue, error) {
	template = strings.TrimSpace(template)
	scanner := NewXScanner(strings.NewReader(template), allowedTopLevels)

	// parse our first token
	tokenType, token := scanner.Scan()

	// try to scan to our next token
	nextTT, _ := scanner.Scan()

	// if we only have an identifier or an expression, evaluate it on its own
	if nextTT == EOF {
		switch tokenType {
		case IDENTIFIER:
			return ResolveValue(env, context, token), nil
		case EXPRESSION:
			return EvaluateExpression(env, context, token), nil
		}
	}

	// otherwise fallback to full template evaluation
	asStr, err := EvaluateTemplateAsString(env, context, template, allowedTopLevels)
	return types.NewXText(asStr), err
}

// EvaluateTemplateAsString evaluates the passed in template returning the string value of its execution
func EvaluateTemplateAsString(env utils.Environment, context types.XValue, template string, allowedTopLevels []string) (string, error) {
	var buf bytes.Buffer
	scanner := NewXScanner(strings.NewReader(template), allowedTopLevels)
	errors := NewTemplateErrors()

	for tokenType, token := scanner.Scan(); tokenType != EOF; tokenType, token = scanner.Scan() {
		switch tokenType {
		case BODY:
			buf.WriteString(token)
		case IDENTIFIER:
			value := ResolveValue(env, context, token)

			if types.IsXError(value) {
				errors.Add(fmt.Sprintf("@%s", token), value.(error).Error())
			} else {
				strValue, _ := types.ToXText(env, value)

				buf.WriteString(strValue.Native())
			}
		case EXPRESSION:
			value := EvaluateExpression(env, context, token)

			if types.IsXError(value) {
				errors.Add(fmt.Sprintf("@(%s)", token), value.(error).Error())
			} else {
				strValue, _ := types.ToXText(env, value)

				buf.WriteString(strValue.Native())
			}
		}
	}

	if errors.HasErrors() {
		return buf.String(), errors
	}
	return buf.String(), nil
}

func indexInto(env utils.Environment, variable types.XValue, index types.XNumber) types.XValue {
	indexable, isIndexable := variable.(types.XIndexable)
	if !isIndexable {
		return types.NewXErrorf("%s is not indexable", variable.Describe())
	}

	indexAsInt, xerr := types.ToInteger(env, index)
	if xerr != nil {
		return xerr
	}

	if indexAsInt >= indexable.Length() || indexAsInt < -indexable.Length() {
		return types.NewXErrorf("index %d out of range for %d items", indexAsInt, indexable.Length())
	}
	if indexAsInt < 0 {
		indexAsInt += indexable.Length()
	}
	return indexable.Index(indexAsInt)
}

// ResolveValue will resolve the passed in string variable given in dot notation and return
// the value as defined by the Resolvable passed in.
func ResolveValue(env utils.Environment, variable types.XValue, key string) types.XValue {
	rest := key
	for rest != "" {
		key, rest = popNextVariable(rest)

		if utils.IsNil(variable) {
			return types.NewXErrorf("%s has no property '%s'", types.Describe(variable), key)
		}

		// is our key numeric?
		index, err := strconv.Atoi(key)
		if err == nil {
			variable = indexInto(env, variable, types.NewXNumberFromInt(index))
			if types.IsXError(variable) {
				return variable
			}
			continue
		}

		resolver, isResolver := variable.(types.XResolvable)

		// look it up in our resolver
		if isResolver {
			variable = resolver.Resolve(env, key)

			if types.IsXError(variable) {
				return variable
			}

		} else {
			return types.NewXErrorf("%s has no property '%s'", types.Describe(variable), key)
		}
	}

	return variable
}

// popNextVariable pops the next variable off our string:
//     foo.bar.baz -> "foo", "bar.baz"
//     foo.0.bar -> "foo", "0.baz"
func popNextVariable(input string) (string, string) {
	var keyStart = 0
	var keyEnd = -1
	var restStart = -1

	for i, c := range input {
		if c == '.' {
			keyEnd = i
			restStart = i + 1
			break
		}
	}

	if keyEnd == -1 {
		return input, ""
	}

	key := strings.Trim(input[keyStart:keyEnd], "\"")
	rest := input[restStart:]

	return key, rest
}
