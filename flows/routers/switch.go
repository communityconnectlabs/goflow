package routers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/envs"
	"github.com/greatnonprofits-nfp/goflow/excellent/types"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/inspect"
	"github.com/greatnonprofits-nfp/goflow/flows/routers/cases"
	"github.com/greatnonprofits-nfp/goflow/utils"
	"github.com/greatnonprofits-nfp/goflow/utils/jsonx"
	"github.com/greatnonprofits-nfp/goflow/utils/uuids"

	"github.com/pkg/errors"
	"strconv"
	"net/http"
	"io/ioutil"
	"net/url"
)

func init() {
	registerType(TypeSwitch, readSwitchRouter)
}

// TypeSwitch is the constant for our switch router
const TypeSwitch string = "switch"

// Case represents a single case and test in our switch
type Case struct {
	UUID         uuids.UUID         `json:"uuid"                   validate:"required"`
	Type         string             `json:"type"                   validate:"required"`
	Arguments    []string           `json:"arguments,omitempty"    engine:"localized,evaluated"`
	CategoryUUID flows.CategoryUUID `json:"category_uuid"          validate:"required"`
}

// NewCase creates a new case
func NewCase(uuid uuids.UUID, type_ string, arguments []string, categoryUUID flows.CategoryUUID) *Case {
	return &Case{
		UUID:         uuid,
		Type:         type_,
		Arguments:    arguments,
		CategoryUUID: categoryUUID,
	}
}

// LocalizationUUID gets the UUID which identifies this object for localization
func (c *Case) LocalizationUUID() uuids.UUID { return uuids.UUID(c.UUID) }

func (c *Case) Dependencies(localization flows.Localization, include func(envs.Language, assets.Reference)) {
	groupRef := func(args []string) assets.Reference {
		// if we have two args, the second is name
		name := ""
		if len(args) == 2 {
			name = args[1]
		}
		return assets.NewGroupReference(assets.GroupUUID(args[0]), name)
	}

	// currently only the HAS_GROUP router test can produce a dependency
	if c.Type == "has_group" && len(c.Arguments) > 0 {
		include(envs.NilLanguage, groupRef(c.Arguments))

		// the group UUID might be different in different translations
		for _, lang := range localization.Languages() {
			arguments := localization.GetTranslations(lang).GetTextArray(c.UUID, "arguments")
			if len(arguments) > 0 {
				include(lang, groupRef(arguments))
			}
		}
	}
}

// SwitchRouter is a router which allows specifying 0-n cases which should each be tested in order, following
// whichever case returns true, or if none do, then taking the default category
type SwitchRouter struct {
	baseRouter

	operand  string
	cases    []*Case
	default_ flows.CategoryUUID
	config   *SwitchRouterConfig
}

// NewSwitch creates a new switch router
func NewSwitch(wait flows.Wait, resultName string, categories []*Category, operand string, cases []*Case, defaultCategory flows.CategoryUUID, config *SwitchRouterConfig) *SwitchRouter {
	return &SwitchRouter{
		baseRouter: newBaseRouter(TypeSwitch, wait, resultName, categories),
		default_:   defaultCategory,
		operand:    operand,
		cases:      cases,
		config:     config,
	}
}

func (r *SwitchRouter) Cases() []*Case { return r.cases }

// Validate validates the arguments for this router
func (r *SwitchRouter) Validate(exits []flows.Exit) error {
	// check the default category is valid
	if r.default_ != "" && !r.isValidCategory(r.default_) {
		return errors.Errorf("default category %s is not a valid category", r.default_)
	}

	for _, c := range r.cases {
		// check each case points to a valid category
		if !r.isValidCategory(c.CategoryUUID) {
			return errors.Errorf("case category %s is not a valid category", c.CategoryUUID)
		}

		// and each case test is valid
		if _, exists := cases.XTESTS[c.Type]; !exists {
			return errors.Errorf("case test %s is not a registered test function", c.Type)
		}
	}

	return r.validate(exits)
}

// Route determines which exit to take from a node
func (r *SwitchRouter) Route(run flows.FlowRun, step flows.Step, logEvent flows.EventCallback) (flows.ExitUUID, error) {
	env := run.Environment()

	if r.operand == "@input.text" {
		r.operand = "@input"
	}

	// first evaluate our operand
	operand, err := run.EvaluateTemplateValue(r.operand)
	if err != nil {
		run.LogError(step, err)
	}

	var input string
	var corrected string

	if operand != nil {
		asText, _ := types.ToXText(env, operand)
		input = asText.Native()
		corrected = input

		// It only calls Bing Spell Checker if the text has more than 5 characters
		if r.config.EnabledSpell && len(input) > 5 {
			defaultLangSpellChecker := "en-US"
			spellCheckerLangs := map[string]string{
				"spa": "es-US",
				"vie": "vi",
				"kor": "ko-KR",
				"chi": "zh-hans",
				"por": "pt-BR",
			}
			spellCheckerLangCode := spellCheckerLangs[string(run.Contact().Language())]
			if spellCheckerLangCode == "" {
				spellCheckerLangCode = defaultLangSpellChecker
			}

			sensitivityConfig, _ := strconv.ParseFloat(r.config.SpellSensitivity, 32)
			spellingCorrectionSensitivity := sensitivityConfig / 100

			spellCheckerAPIKey := utils.GetEnv(utils.MailroomSpellCheckerKey, "")

			spellCheckerURL := "https://api.cognitive.microsoft.com/bing/v7.0/SpellCheck/"
			spellCheckerPayloadpayload := url.Values{}
			spellCheckerPayloadpayload.Add("text", input)
			spellCheckerPayloadpayload.Add("mkt", spellCheckerLangCode)
			spellCheckerPayloadpayload.Add("mode", "spell")

			spellCheckerURL = fmt.Sprintf("%s?%s", spellCheckerURL, spellCheckerPayloadpayload.Encode())
			spellCheckerReq, _ := http.NewRequest("GET", spellCheckerURL, strings.NewReader(""))
			spellCheckerReq.Header.Add("Ocp-Apim-Subscription-Key", spellCheckerAPIKey)

			resp, _ := http.DefaultClient.Do(spellCheckerReq)
			defer resp.Body.Close()

			content, _ := ioutil.ReadAll(resp.Body)

			var bodyResp SpellCheckerPayload
			err = json.Unmarshal(content, &bodyResp)

			if resp.StatusCode == 200 && err == nil {
				flaggedTokens := bodyResp.FlaggedTokens
				for _, token := range flaggedTokens {
					for _, suggestion := range token.Suggestions {
						if suggestion.Score >= spellingCorrectionSensitivity {
							corrected = strings.Replace(corrected, token.Token, suggestion.Suggestion, -1)
						}
					}
				}
			}
		}
	}

	// find first matching case
	match, categoryUUID, extra, err := r.matchCase(run, step, operand, corrected)
	if err != nil {
		return "", err
	}

	// none of our cases matched, so try to use the default
	if categoryUUID == "" && r.default_ != "" {
		// evaluate our operand as a string
		value, xerr := types.ToXText(env, operand)
		if xerr != nil {
			run.LogError(step, xerr)
		}

		match = value.Native()
		categoryUUID = r.default_
	}

	return r.routeToCategory(run, step, categoryUUID, match, input, extra, logEvent, corrected)
}

func (r *SwitchRouter) matchCase(run flows.FlowRun, step flows.Step, operand types.XValue, corrected string) (string, flows.CategoryUUID, *types.XObject, error) {
	for _, c := range r.cases {
		test := strings.ToLower(c.Type)

		// try to look up our function
		xtest := cases.XTESTS[test]
		if xtest == nil {
			return "", "", nil, errors.Errorf("unknown case test '%s'", c.Type)
		}

		// build our argument list which starts with the operand
		args := []types.XValue{operand}

		localizedArgs := run.GetTextArray(c.UUID, "arguments", c.Arguments)
		for i := range c.Arguments {
			test := localizedArgs[i]
			arg, err := run.EvaluateTemplateValue(test)
			if err != nil {
				run.LogError(step, err)
			}
			fmt.Printf("%s \n", arg)
			fmt.Printf("%s \n", test)
			args = append(args, arg)
		}

		// call our function
		result := xtest(run.Environment(), args...)

		// tests have to return either errors or test results
		switch typed := result.(type) {
		case types.XError:
			// test functions can return an error
			run.LogError(step, errors.Errorf("error calling test %s: %s", strings.ToUpper(test), typed.Error()))
		case *types.XObject:
			matched := typed.Truthy()
			if !matched {
				continue
			}

			match, _ := typed.Get("match")
			extra, _ := typed.Get("extra")

			extraAsObject, isObject := extra.(*types.XObject)
			if extra != nil && !isObject {
				run.LogError(step, errors.Errorf("test %s returned non-object extra", strings.ToUpper(test)))
			}

			resultAsStr, xerr := types.ToXText(run.Environment(), match)
			if xerr != nil {
				return "", "", nil, xerr
			}

			return resultAsStr.Native(), c.CategoryUUID, extraAsObject, nil
		default:
			panic(fmt.Sprintf("unexpected result type from test %v: %#v", xtest, result))
		}
	}
	return "", "", nil, nil
}

// EnumerateTemplates enumerates all expressions on this object and its children
func (r *SwitchRouter) EnumerateTemplates(localization flows.Localization, include func(envs.Language, string)) {
	include(envs.NilLanguage, r.operand)

	inspect.Templates(r.cases, localization, include)
}

// EnumerateDependencies enumerates all dependencies on this object and its children
func (r *SwitchRouter) EnumerateDependencies(localization flows.Localization, include func(envs.Language, assets.Reference)) {
	inspect.Dependencies(r.cases, localization, include)
}

//------------------------------------------------------------------------------------------
// JSON Encoding / Decoding
//------------------------------------------------------------------------------------------

type SwitchRouterConfig struct {
	EnabledSpell     bool   `json:"spell_checker"`
	SpellSensitivity string `json:"spelling_correction_sensitivity"`
}

type SpellCheckerPayload struct {
	Type           string                     `json:"_type"`
	FlaggedTokens  []SpellCheckerFlaggedToken `json:"flaggedTokens"`
	CorrectionType string                     `json:"correctionType"`
}

type SpellCheckerFlaggedToken struct {
	Offset      int                      `json:"offset"`
	Token       string                   `json:"token"`
	Type        string                   `json:"type"`
	Suggestions []SpellCheckerSuggestion `json:"suggestions"`
}

type SpellCheckerSuggestion struct {
	Suggestion string  `json:"suggestion"`
	Score      float64 `json:"score"`
}

type switchRouterEnvelope struct {
	baseRouterEnvelope

	Operand string              `json:"operand"               validate:"required"`
	Cases   []*Case             `json:"cases"`
	Default flows.CategoryUUID  `json:"default_category_uuid" validate:"omitempty,uuid4"`
	Config  *SwitchRouterConfig `json:"config"`
}

func readSwitchRouter(data json.RawMessage) (flows.Router, error) {
	e := &switchRouterEnvelope{}
	if err := utils.UnmarshalAndValidate(data, e); err != nil {
		return nil, err
	}

	r := &SwitchRouter{
		operand:  e.Operand,
		cases:    e.Cases,
		default_: e.Default,
		config:   e.Config,
	}

	if err := r.unmarshal(&e.baseRouterEnvelope); err != nil {
		return nil, err
	}

	return r, nil
}

// MarshalJSON marshals this resume into JSON
func (r *SwitchRouter) MarshalJSON() ([]byte, error) {
	e := &switchRouterEnvelope{
		Operand: r.operand,
		Cases:   r.cases,
		Default: r.default_,
		Config:  r.config,
	}

	if err := r.marshal(&e.baseRouterEnvelope); err != nil {
		return nil, err
	}

	return jsonx.Marshal(e)
}
