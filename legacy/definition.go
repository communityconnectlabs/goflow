package legacy

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/extensions/transferto"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/actions"
	"github.com/greatnonprofits-nfp/goflow/flows/definition"
	"github.com/greatnonprofits-nfp/goflow/flows/routers"
	"github.com/greatnonprofits-nfp/goflow/flows/routers/waits"
	"github.com/greatnonprofits-nfp/goflow/flows/routers/waits/hints"
	"github.com/greatnonprofits-nfp/goflow/legacy/expressions"
	"github.com/greatnonprofits-nfp/goflow/utils"
	"github.com/nyaruka/gocommon/urns"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

// Flow is a flow in the legacy format
type Flow struct {
	BaseLanguage utils.Language `json:"base_language"`
	FlowType     string         `json:"flow_type"`
	RuleSets     []RuleSet      `json:"rule_sets" validate:"dive"`
	ActionSets   []ActionSet    `json:"action_sets" validate:"dive"`
	Entry        flows.NodeUUID `json:"entry" validate:"omitempty,uuid4"`
	Metadata     *Metadata      `json:"metadata"`

	// some flows have these set here instead of in metadata
	UUID assets.FlowUUID `json:"uuid"`
	Name string          `json:"name"`
}

// Metadata is the metadata section of a legacy flow
type Metadata struct {
	UUID     assets.FlowUUID `json:"uuid"`
	Name     string          `json:"name"`
	Revision int             `json:"revision"`
	Expires  int             `json:"expires"`
	Notes    []Note          `json:"notes,omitempty"`
}

type Rule struct {
	UUID            flows.ExitUUID `json:"uuid" validate:"required,uuid4"`
	Destination     flows.NodeUUID `json:"destination" validate:"omitempty,uuid4"`
	DestinationType string         `json:"destination_type" validate:"eq=A|eq=R"`
	Test            TypedEnvelope  `json:"test"`
	Category        Translations   `json:"category"`
}

type RuleSet struct {
	Y           int             `json:"y"`
	X           int             `json:"x"`
	UUID        flows.NodeUUID  `json:"uuid" validate:"required,uuid4"`
	Type        string          `json:"ruleset_type"`
	Label       string          `json:"label"`
	Operand     string          `json:"operand"`
	Rules       []Rule          `json:"rules"`
	Config      json.RawMessage `json:"config"`
	FinishedKey string          `json:"finished_key"`
}

type ActionSet struct {
	Y           int            `json:"y"`
	X           int            `json:"x"`
	Destination flows.NodeUUID `json:"destination" validate:"omitempty,uuid4"`
	ExitUUID    flows.ExitUUID `json:"exit_uuid" validate:"required,uuid4"`
	UUID        flows.NodeUUID `json:"uuid" validate:"required,uuid4"`
	Actions     []Action       `json:"actions"`
}

type LabelReference struct {
	UUID assets.LabelUUID
	Name string
}

func (l *LabelReference) Migrate() *assets.LabelReference {
	if len(l.UUID) > 0 {
		return assets.NewLabelReference(l.UUID, l.Name)
	}
	return assets.NewVariableLabelReference(l.Name)
}

// UnmarshalJSON unmarshals a legacy label reference from the given JSON
func (l *LabelReference) UnmarshalJSON(data []byte) error {
	// label reference may be a string
	if data[0] == '"' {
		var nameExpression string
		if err := json.Unmarshal(data, &nameExpression); err != nil {
			return err
		}

		// if it starts with @ then it's an expression
		if strings.HasPrefix(nameExpression, "@") {
			nameExpression, _ = expressions.MigrateTemplate(nameExpression, nil)
		}

		l.Name = nameExpression
		return nil
	}

	// or a JSON object with UUID/Name properties
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	l.UUID = assets.LabelUUID(raw["uuid"])
	l.Name = raw["name"]
	return nil
}

type ContactReference struct {
	UUID flows.ContactUUID `json:"uuid"`
	Name string            `json:"name"`
}

func (c *ContactReference) Migrate() *flows.ContactReference {
	return flows.NewContactReference(c.UUID, c.Name)
}

type GroupReference struct {
	UUID assets.GroupUUID
	Name string
}

func (g *GroupReference) Migrate() *assets.GroupReference {
	if len(g.UUID) > 0 {
		return assets.NewGroupReference(g.UUID, g.Name)
	}
	return assets.NewVariableGroupReference(g.Name)
}

// UnmarshalJSON unmarshals a legacy group reference from the given JSON
func (g *GroupReference) UnmarshalJSON(data []byte) error {
	// group reference may be a string
	if data[0] == '"' {
		var nameExpression string
		if err := json.Unmarshal(data, &nameExpression); err != nil {
			return err
		}

		// if it starts with @ then it's an expression
		if strings.HasPrefix(nameExpression, "@") {
			nameExpression, _ = expressions.MigrateTemplate(nameExpression, nil)
		}

		g.Name = nameExpression
		return nil
	}

	// or a JSON object with UUID/Name properties
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	g.UUID = assets.GroupUUID(raw["uuid"])
	g.Name = raw["name"]
	return nil
}

type VariableReference struct {
	ID string `json:"id"`
}

type FlowReference struct {
	UUID assets.FlowUUID `json:"uuid"`
	Name string          `json:"name"`
}

func (f *FlowReference) Migrate() *assets.FlowReference {
	return assets.NewFlowReference(f.UUID, f.Name)
}

// RulesetConfig holds the config dictionary for a legacy ruleset
type RulesetConfig struct {
	Flow           *assets.FlowReference `json:"flow"`
	FieldDelimiter string                `json:"field_delimiter"`
	FieldIndex     int                   `json:"field_index"`
	Webhook        string                `json:"webhook"`
	WebhookAction  string                `json:"webhook_action"`
	WebhookHeaders []WebhookHeader       `json:"webhook_headers"`
	Resthook       string                `json:"resthook"`
	LookupDb       map[string]string     `json:"lookup_db"`
	LookupQueries  []actions.LookupQuery `json:"lookup_queries"`
}

type WebhookHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Action struct {
	Type string           `json:"type"`
	UUID flows.ActionUUID `json:"uuid"`
	Name string           `json:"name"`

	// message and email
	Msg          json.RawMessage `json:"msg"`
	Media        json.RawMessage `json:"media"`
	QuickReplies json.RawMessage `json:"quick_replies"`
	SendAll      bool            `json:"send_all"`

	// variable contact actions
	Contacts  []ContactReference  `json:"contacts"`
	Groups    []GroupReference    `json:"groups"`
	Variables []VariableReference `json:"variables"`

	// save actions
	Field string `json:"field"`
	Value string `json:"value"`
	Label string `json:"label"`

	// set language
	Language utils.Language `json:"lang"`

	// add label action
	Labels []LabelReference `json:"labels"`

	// start/trigger flow
	Flow FlowReference `json:"flow"`

	// channel
	Channel assets.ChannelUUID `json:"channel"`

	// email
	Emails  []string `json:"emails"`
	Subject string   `json:"subject"`

	// IVR
	Recording json.RawMessage `json:"recording"`
	URL       string          `json:"url"`
}

type subflowTest struct {
	ExitType string `json:"exit_type"`
}

type webhookTest struct {
	Status string `json:"status"`
}

type airtimeTest struct {
	ExitStatus string `json:"exit_status"`
}

type localizedStringTest struct {
	Test Translations `json:"test"`
}

type stringTest struct {
	Test string `json:"test"`
}

type numericTest struct {
	Test StringOrNumber `json:"test"`
}

type betweenTest struct {
	Min string `json:"min"`
	Max string `json:"max"`
}

type timeoutTest struct {
	Minutes int `json:"minutes"`
}

type groupTest struct {
	Test GroupReference `json:"test"`
}

type wardTest struct {
	State    string `json:"state"`
	District string `json:"district"`
}

var relativeDateTest = regexp.MustCompile(`@\(date\.today\s+(\+|\-)\s+(\d+)\)`)

var flowTypeMapping = map[string]flows.FlowType{
	"":  flows.FlowTypeMessaging, // some campaign event flows are missing this
	"F": flows.FlowTypeMessaging,
	"M": flows.FlowTypeMessaging,
	"V": flows.FlowTypeVoice,
	"S": flows.FlowTypeMessagingOffline,
}

func addTranslationMap(baseLanguage utils.Language, localization flows.Localization, mapped Translations, uuid utils.UUID, property string) string {
	var inBaseLanguage string
	for language, item := range mapped {
		expression, _ := expressions.MigrateTemplate(item, nil)
		if language != baseLanguage && language != "base" {
			localization.AddItemTranslation(language, uuid, property, []string{expression})
		} else {
			inBaseLanguage = expression
		}
	}

	return inBaseLanguage
}

func addTranslationMultiMap(baseLanguage utils.Language, localization flows.Localization, mapped map[utils.Language][]string, uuid utils.UUID, property string) []string {
	var inBaseLanguage []string
	for language, items := range mapped {
		templates := make([]string, len(items))
		for i := range items {
			expression, _ := expressions.MigrateTemplate(items[i], nil)
			templates[i] = expression
		}
		if language != baseLanguage {
			localization.AddItemTranslation(language, uuid, property, templates)
		} else {
			inBaseLanguage = templates
		}
	}
	return inBaseLanguage
}

// TransformTranslations transforms a list of single item translations into a map of multi-item translations, e.g.
//
// [{"eng": "yes", "fra": "oui"}, {"eng": "no", "fra": "non"}] becomes {"eng": ["yes", "no"], "fra": ["oui", "non"]}
//
func TransformTranslations(items []Translations) map[utils.Language][]string {
	// re-organize into a map of arrays
	transformed := make(map[utils.Language][]string)

	for i := range items {
		for language, translation := range items[i] {
			perLanguage, found := transformed[language]
			if !found {
				perLanguage = make([]string, len(items))
				transformed[language] = perLanguage
			}
			perLanguage[i] = translation
		}
	}
	return transformed
}

var testTypeMappings = map[string]string{
	"between":              "has_number_between",
	"contains":             "has_all_words",
	"contains_any":         "has_any_word",
	"contains_only_phrase": "has_only_phrase",
	"contains_phrase":      "has_phrase",
	"date":                 "has_date",
	"date_after":           "has_date_gt",
	"date_before":          "has_date_lt",
	"date_equal":           "has_date_eq",
	"district":             "has_district",
	"has_email":            "has_email",
	"eq":                   "has_number_eq",
	"gt":                   "has_number_gt",
	"gte":                  "has_number_gte",
	"in_group":             "has_group",
	"lt":                   "has_number_lt",
	"lte":                  "has_number_lte",
	"not_empty":            "has_text",
	"number":               "has_number",
	"phone":                "has_phone",
	"regex":                "has_pattern",
	"starts":               "has_beginning",
	"state":                "has_state",
	"ward":                 "has_ward",
}

// migrates the given legacy action to a new action
func migrateAction(baseLanguage utils.Language, a Action, localization flows.Localization, baseMediaURL string) (flows.Action, error) {
	switch a.Type {
	case "add_label":
		labels := make([]*assets.LabelReference, len(a.Labels))
		for i, label := range a.Labels {
			labels[i] = label.Migrate()
		}

		return actions.NewAddInputLabelsAction(a.UUID, labels), nil

	case "email":
		var msg string
		err := json.Unmarshal(a.Msg, &msg)
		if err != nil {
			return nil, err
		}

		migratedSubject, _ := expressions.MigrateTemplate(a.Subject, nil)
		migratedBody, _ := expressions.MigrateTemplate(msg, nil)
		migratedEmails := make([]string, len(a.Emails))
		for i, email := range a.Emails {
			migratedEmails[i], _ = expressions.MigrateTemplate(email, nil)
		}

		return actions.NewSendEmailAction(a.UUID, migratedEmails, migratedSubject, migratedBody), nil

	case "lang":
		return actions.NewSetContactLanguageAction(a.UUID, string(a.Language)), nil
	case "channel":
		return actions.NewSetContactChannelAction(a.UUID, assets.NewChannelReference(a.Channel, a.Name)), nil
	case "flow":
		return actions.NewEnterFlowAction(a.UUID, a.Flow.Migrate(), true), nil
	case "trigger-flow":
		contacts := make([]*flows.ContactReference, len(a.Contacts))
		for i, contact := range a.Contacts {
			contacts[i] = contact.Migrate()
		}
		groups := make([]*assets.GroupReference, len(a.Groups))
		for i, group := range a.Groups {
			groups[i] = group.Migrate()
		}
		var createContact bool
		variables := make([]string, 0, len(a.Variables))
		for _, variable := range a.Variables {
			if variable.ID == "@new_contact" {
				createContact = true
			} else {
				migratedVar, _ := expressions.MigrateTemplate(variable.ID, nil)
				variables = append(variables, migratedVar)
			}
		}

		return actions.NewStartSessionAction(a.UUID, a.Flow.Migrate(), []urns.URN{}, contacts, groups, variables, createContact), nil
	case "reply", "send":
		media := make(Translations)
		var quickReplies map[utils.Language][]string

		msg, err := ReadTranslations(a.Msg)
		if err != nil {
			return nil, err
		}

		if a.Media != nil {
			err := json.Unmarshal(a.Media, &media)
			if err != nil {
				return nil, err
			}
		}
		if a.QuickReplies != nil {
			legacyQuickReplies := make([]Translations, 0)

			err := json.Unmarshal(a.QuickReplies, &legacyQuickReplies)
			if err != nil {
				return nil, err
			}

			quickReplies = TransformTranslations(legacyQuickReplies)
		}

		for lang, attachment := range media {
			parts := strings.SplitN(attachment, ":", 2)
			var mediaType, mediaURL string
			if len(parts) == 2 {
				mediaType = parts[0]
				mediaURL = parts[1]
			} else {
				// no media type defaults to image
				mediaType = "image"
				mediaURL = parts[0]
			}

			// attachment is a real upload and not just an expression, need to make it absolute
			if !strings.Contains(mediaURL, "@") {
				media[lang] = fmt.Sprintf("%s:%s", mediaType, URLJoin(baseMediaURL, mediaURL))
			}
		}

		migratedText := addTranslationMap(baseLanguage, localization, msg, utils.UUID(a.UUID), "text")
		migratedMedia := addTranslationMap(baseLanguage, localization, media, utils.UUID(a.UUID), "attachments")
		migratedQuickReplies := addTranslationMultiMap(baseLanguage, localization, quickReplies, utils.UUID(a.UUID), "quick_replies")

		attachments := []string{}
		if migratedMedia != "" {
			attachments = append(attachments, migratedMedia)
		}

		if a.Type == "reply" {
			return actions.NewSendMsgAction(a.UUID, migratedText, attachments, migratedQuickReplies, a.SendAll), nil
		}

		contacts := make([]*flows.ContactReference, len(a.Contacts))
		for i, contact := range a.Contacts {
			contacts[i] = contact.Migrate()
		}
		groups := make([]*assets.GroupReference, len(a.Groups))
		for i, group := range a.Groups {
			groups[i] = group.Migrate()
		}
		variables := make([]string, 0, len(a.Variables))
		for _, variable := range a.Variables {
			migratedVar, _ := expressions.MigrateTemplate(variable.ID, nil)
			variables = append(variables, migratedVar)
		}

		return actions.NewSendBroadcastAction(a.UUID, migratedText, attachments, migratedQuickReplies, []urns.URN{}, contacts, groups, variables), nil

	case "add_group":
		groups := make([]*assets.GroupReference, len(a.Groups))
		for i, group := range a.Groups {
			groups[i] = group.Migrate()
		}

		return actions.NewAddContactGroupsAction(a.UUID, groups), nil
	case "del_group":
		groups := make([]*assets.GroupReference, len(a.Groups))
		for i, group := range a.Groups {
			groups[i] = group.Migrate()
		}

		allGroups := len(groups) == 0
		return actions.NewRemoveContactGroupsAction(a.UUID, groups, allGroups), nil
	case "save":
		migratedValue, _ := expressions.MigrateTemplate(a.Value, nil)

		// flows now have different action for name changing
		if a.Field == "name" || a.Field == "first_name" {
			// we can emulate setting only the first name with an expression
			if a.Field == "first_name" {
				migratedValue = strings.TrimSpace(migratedValue)
				migratedValue = fmt.Sprintf("%s @(word_slice(contact.name, 1, -1))", migratedValue)
			}

			return actions.NewSetContactNameAction(a.UUID, migratedValue), nil
		}

		// and another new action for adding a URN
		if urns.IsValidScheme(a.Field) {
			return actions.NewAddContactURNAction(a.UUID, a.Field, migratedValue), nil
		} else if a.Field == "tel_e164" {
			return actions.NewAddContactURNAction(a.UUID, "tel", migratedValue), nil
		}

		return actions.NewSetContactFieldAction(a.UUID, assets.NewFieldReference(a.Field, a.Label), migratedValue), nil
	case "say":
		msg, err := ReadTranslations(a.Msg)
		if err != nil {
			return nil, err
		}
		recording, err := ReadTranslations(a.Recording)
		if err != nil {
			return nil, err
		}

		// make audio URLs absolute
		for lang, audioURL := range recording {
			if audioURL != "" {
				recording[lang] = URLJoin(baseMediaURL, audioURL)
			}
		}

		migratedText := addTranslationMap(baseLanguage, localization, msg, utils.UUID(a.UUID), "text")
		migratedAudioURL := addTranslationMap(baseLanguage, localization, recording, utils.UUID(a.UUID), "audio_url")

		return actions.NewSayMsgAction(a.UUID, migratedText, migratedAudioURL), nil
	case "play":
		// note this URL is already assumed to be absolute
		migratedAudioURL, _ := expressions.MigrateTemplate(a.URL, nil)

		return actions.NewPlayAudioAction(a.UUID, migratedAudioURL), nil
	default:
		return nil, errors.Errorf("unable to migrate legacy action type: %s", a.Type)
	}
}

// migrates the given legacy rulset to a node with a router
func migrateRuleSet(lang utils.Language, r RuleSet, validDests map[flows.NodeUUID]bool, localization flows.Localization) (flows.Node, UINodeType, NodeUIConfig, error) {
	var newActions []flows.Action
	var router flows.Router
	var wait flows.Wait
	var uiType UINodeType
	uiConfig := make(NodeUIConfig)

	cases, categories, defaultCategory, timeoutCategory, exits, err := migrateRules(lang, r, validDests, localization, uiConfig)
	if err != nil {
		return nil, "", nil, err
	}

	resultName := r.Label

	// load the config for this ruleset
	var config RulesetConfig
	if r.Config != nil {
		err := json.Unmarshal(r.Config, &config)
		if err != nil {
			return nil, "", nil, err
		}
	}

	// sometimes old flows don't have this set
	if r.Type == "" {
		r.Type = "wait_message"
	}

	switch r.Type {
	case "subflow":
		newActions = []flows.Action{
			actions.NewEnterFlowAction(flows.ActionUUID(utils.NewUUID()), config.Flow, false),
		}

		// subflow rulesets operate on the child flow status
		router = routers.NewSwitchRouter(nil, resultName, categories, "@child.run.status", cases, defaultCategory)
		uiType = UINodeTypeSplitBySubflow

	case "webhook":
		migratedURL, _ := expressions.MigrateTemplate(config.Webhook, &expressions.MigrateOptions{URLEncode: true})
		headers := make(map[string]string, len(config.WebhookHeaders))
		body := ""
		method := strings.ToUpper(config.WebhookAction)
		if method == "" {
			method = "POST"
		}

		if method == "POST" {
			headers["Content-Type"] = "application/json"
			body = flows.LegacyWebhookPayload
		}

		for _, header := range config.WebhookHeaders {
			// ignore empty headers sometimes left in flow definitions
			if header.Name != "" {
				headers[header.Name], _ = expressions.MigrateTemplate(header.Value, nil)
			}
		}

		newActions = []flows.Action{
			actions.NewCallWebhookAction(flows.ActionUUID(utils.NewUUID()), method, migratedURL, headers, body, resultName),
		}

		// webhook rulesets operate on the webhook status, saved as category
		operand := fmt.Sprintf("@results.%s.category", utils.Snakify(resultName))
		router = routers.NewSwitchRouter(nil, "", categories, operand, cases, defaultCategory)
		uiType = UINodeTypeSplitByWebhook

	case "lookup":
		for i := range config.LookupQueries {
			// ignore empty values sometimes left in flow definitions
			if config.LookupQueries[i].Value != "" {
				config.LookupQueries[i].Value, _ = expressions.MigrateTemplate(config.LookupQueries[i].Value, nil)
			}
		}

		newActions = []flows.Action{
			actions.NewCallLookupAction(flows.ActionUUID(utils.NewUUID()), config.LookupDb, config.LookupQueries, resultName),
		}

		// lookup rulesets operate on the webhook status, saved as category
		operand := fmt.Sprintf("@results.%s.category", utils.Snakify(resultName))
		router = routers.NewSwitchRouter(nil, "", categories, operand, cases, defaultCategory)
		uiType = UINodeTypeSplitByLookup

	case "resthook":
		newActions = []flows.Action{
			actions.NewCallResthookAction(flows.ActionUUID(utils.NewUUID()), config.Resthook, resultName),
		}

		// resthook rulesets operate on the webhook status, saved as category
		operand := fmt.Sprintf("@results.%s.category", utils.Snakify(resultName))
		router = routers.NewSwitchRouter(nil, "", categories, operand, cases, defaultCategory)
		uiType = UINodeTypeSplitByResthook

	case "form_field":
		operand, _ := expressions.MigrateTemplate(r.Operand, nil)
		operand = fmt.Sprintf("@(field(%s, %d, \"%s\"))", operand[1:], config.FieldIndex, config.FieldDelimiter)
		router = routers.NewSwitchRouter(nil, resultName, categories, operand, cases, defaultCategory)

		lastDot := strings.LastIndex(r.Operand, ".")
		if lastDot > -1 {
			fieldKey := r.Operand[lastDot+1:]

			uiConfig["operand"] = map[string]string{"id": fieldKey}
			uiConfig["delimiter"] = config.FieldDelimiter
			uiConfig["index"] = config.FieldIndex
		}

		uiType = UINodeTypeSplitByRunResultDelimited

	case "group":
		// in legacy flows these rulesets have their operand as @step.value but it's not used
		router = routers.NewSwitchRouter(nil, resultName, categories, "@contact.groups", cases, defaultCategory)
		uiType = UINodeTypeSplitByGroups

	case "wait_message", "wait_audio", "wait_video", "wait_photo", "wait_gps", "wait_recording", "wait_digit", "wait_digits":
		// look for timeout test on the legacy ruleset
		timeoutSeconds := 0
		for _, rule := range r.Rules {
			if rule.Test.Type == "timeout" {
				test := timeoutTest{}
				if err := json.Unmarshal(rule.Test.Data, &test); err != nil {
					return nil, "", nil, err
				}
				timeoutSeconds = 60 * test.Minutes
				break
			}
		}

		var timeout *waits.Timeout
		if timeoutSeconds > 0 && timeoutCategory != "" {
			timeout = waits.NewTimeout(timeoutSeconds, timeoutCategory)
		}

		wait = waits.NewMsgWait(timeout, migrateRuleSetToHint(r))
		uiType = UINodeTypeWaitForResponse

		fallthrough
	case "flow_field", "contact_field", "expression":
		// unlike other templates, operands for expression rulesets need to be wrapped in such a way that if
		// they error, they evaluate to the original expression
		var defaultToSelf bool
		switch r.Type {
		case "flow_field":
			uiType = UINodeTypeSplitByRunResult
			lastDot := strings.LastIndex(r.Operand, ".")
			if lastDot > -1 {
				fieldKey := r.Operand[lastDot+1:]

				uiConfig["operand"] = map[string]string{"id": fieldKey}
			}
		case "contact_field":
			uiType = UINodeTypeSplitByContactField

			lastDot := strings.LastIndex(r.Operand, ".")
			if lastDot > -1 {
				fieldKey := r.Operand[lastDot+1:]
				if fieldKey == "name" {
					uiConfig["operand"] = map[string]string{
						"type": "property",
						"id":   "name",
						"name": "Name",
					}
				} else if fieldKey == "groups" {
					uiType = UINodeTypeSplitByExpression

				} else if urns.IsValidScheme(fieldKey) {
					uiConfig["operand"] = map[string]string{
						"type": "scheme",
						"id":   fieldKey,
					}
				} else {
					uiConfig["operand"] = map[string]string{
						"type": "field",
						"id":   fieldKey,
					}
				}
			}

		case "expression":
			defaultToSelf = true
			uiType = UINodeTypeSplitByExpression
		}

		operand, _ := expressions.MigrateTemplate(r.Operand, &expressions.MigrateOptions{DefaultToSelf: defaultToSelf})
		if operand == "" {
			operand = "@input"
		}

		router = routers.NewSwitchRouter(wait, resultName, categories, operand, cases, defaultCategory)
	case "random":
		router = routers.NewRandomRouter(nil, resultName, categories)
		uiType = UINodeTypeSplitByRandom

	case "airtime":
		countryConfigs := map[string]struct {
			CurrencyCode string          `json:"currency_code"`
			Amount       decimal.Decimal `json:"amount"`
		}{}
		if err := json.Unmarshal(r.Config, &countryConfigs); err != nil {
			return nil, "", nil, err
		}
		currencyAmounts := make(map[string]decimal.Decimal, len(countryConfigs))
		for _, countryCfg := range countryConfigs {
			// check if we already have a configuration for this currency
			existingAmount, alreadyDefined := currencyAmounts[countryCfg.CurrencyCode]
			if alreadyDefined && existingAmount != countryCfg.Amount {
				return nil, "", nil, errors.Errorf("unable to migrate airtime ruleset with different amounts in same currency")
			}

			currencyAmounts[countryCfg.CurrencyCode] = countryCfg.Amount
		}

		newActions = []flows.Action{
			transferto.NewTransferAirtimeAction(flows.ActionUUID(utils.NewUUID()), currencyAmounts, resultName),
		}

		operand := fmt.Sprintf("@results.%s.category", utils.Snakify(resultName))
		router = routers.NewSwitchRouter(nil, "", categories, operand, cases, defaultCategory)
		uiType = UINodeTypeSplitByAirtime

	default:
		return nil, "", nil, errors.Errorf("unrecognized ruleset type: %s", r.Type)
	}

	return definition.NewNode(r.UUID, newActions, router, exits), uiType, uiConfig, nil
}

func migrateRuleSetToHint(r RuleSet) flows.Hint {
	switch r.Type {
	case "wait_audio":
		return hints.NewAudioHint()
	case "wait_video":
		return hints.NewVideoHint()
	case "wait_photo":
		return hints.NewImageHint()
	case "wait_gps":
		return hints.NewLocationHint()
	case "wait_recording":
		return hints.NewAudioHint()
	case "wait_digit":
		return hints.NewFixedDigitsHint(1)
	case "wait_digits":
		return hints.NewTerminatedDigitsHint(r.FinishedKey)
	}
	return nil
}

type categoryAndExit struct {
	category *routers.Category
	exit     flows.Exit
}

// migrates a set of legacy rules to sets of categories, cases and exits
func migrateRules(baseLanguage utils.Language, r RuleSet, validDests map[flows.NodeUUID]bool, localization flows.Localization, uiConfig NodeUIConfig) ([]*routers.Case, []*routers.Category, flows.CategoryUUID, flows.CategoryUUID, []flows.Exit, error) {
	cases := make([]*routers.Case, 0, len(r.Rules))
	categories := make([]*routers.Category, 0, len(r.Rules))
	exits := make([]flows.Exit, 0, len(r.Rules))

	var defaultCategoryUUID, timeoutCategoryUUID flows.CategoryUUID

	convertedByRuleUUID := make(map[flows.ExitUUID]*categoryAndExit, len(r.Rules))
	convertedByCategoryName := make(map[string]*categoryAndExit, len(r.Rules))

	// create categories and exits from the rules
	for _, rule := range r.Rules {
		baseName := rule.Category.Base(baseLanguage)
		var converted *categoryAndExit

		// check if we have previously created category/exits for this category name
		if rule.Test.Type != "true" {
			converted = convertedByCategoryName[baseName]
		}
		if converted == nil {
			// only set exit destination if it's valid
			var destinationUUID flows.NodeUUID
			if validDests[rule.Destination] {
				destinationUUID = rule.Destination
			}

			// rule UUIDs in legacy flows determine path data, so their UUIDs become the exit UUIDs
			exit := definition.NewExit(rule.UUID, destinationUUID)
			exits = append(exits, exit)

			category := routers.NewCategory(flows.CategoryUUID(utils.NewUUID()), baseName, exit.UUID())
			categories = append(categories, category)

			converted = &categoryAndExit{category, exit}
		}

		convertedByRuleUUID[rule.UUID] = converted
		convertedByCategoryName[baseName] = converted

		addTranslationMap(baseLanguage, localization, rule.Category, utils.UUID(converted.category.UUID()), "name")
	}

	// and then a case for each rule
	for _, rule := range r.Rules {
		converted := convertedByRuleUUID[rule.UUID]

		if rule.Test.Type == "true" {
			// implicit Other rules don't become cases, but instead become the router default
			defaultCategoryUUID = converted.category.UUID()
			continue

		} else if rule.Test.Type == "timeout" {
			// timeout rules become category setting on the wait
			timeoutCategoryUUID = converted.category.UUID()
			continue

		} else if rule.Test.Type == "webhook_status" {
			// default case for a webhook ruleset is the last migrated rule (failure)
			defaultCategoryUUID = converted.category.UUID()
		}

		kase, caseUI, err := migrateRule(baseLanguage, rule, converted.category, localization)
		if err != nil {
			return nil, nil, "", "", nil, err
		}

		if caseUI != nil {
			uiConfig.AddCaseConfig(kase.UUID, caseUI)
		}

		cases = append(cases, kase)
	}

	return cases, categories, defaultCategoryUUID, timeoutCategoryUUID, exits, nil
}

// migrates the given legacy rule to a router case
func migrateRule(baseLanguage utils.Language, r Rule, category *routers.Category, localization flows.Localization) (*routers.Case, map[string]interface{}, error) {
	newType, _ := testTypeMappings[r.Test.Type]
	var arguments []string
	var err error

	caseUUID := utils.UUID(utils.NewUUID())
	var caseUI map[string]interface{}

	switch r.Test.Type {

	// tests that take no arguments
	case "date", "has_email", "not_empty", "number", "phone", "state":
		arguments = []string{}

	// tests against a single numeric value
	case "eq", "gt", "gte", "lt", "lte":
		test := numericTest{}
		err = json.Unmarshal(r.Test.Data, &test)
		if err != nil {
			return nil, nil, err
		}
		migratedTest, _ := expressions.MigrateTemplate(string(test.Test), nil)
		arguments = []string{migratedTest}

	case "between":
		test := betweenTest{}
		err = json.Unmarshal(r.Test.Data, &test)
		if err != nil {
			return nil, nil, err
		}

		migratedMin, _ := expressions.MigrateTemplate(test.Min, nil)
		migratedMax, _ := expressions.MigrateTemplate(test.Max, nil)

		arguments = []string{migratedMin, migratedMax}

	// tests against a single localized string
	case "contains", "contains_any", "contains_phrase", "contains_only_phrase", "regex", "starts":
		test := localizedStringTest{}
		err = json.Unmarshal(r.Test.Data, &test)
		if err != nil {
			return nil, nil, err
		}

		baseTest := test.Test.Base(baseLanguage)

		// all the tests are evaluated as templates.. except regex
		if r.Test.Type != "regex" {
			baseTest, _ = expressions.MigrateTemplate(baseTest, nil)

		}
		arguments = []string{baseTest}

		addTranslationMap(baseLanguage, localization, test.Test, caseUUID, "arguments")

	// tests against a single date value
	case "date_equal", "date_after", "date_before":
		test := stringTest{}
		err = json.Unmarshal(r.Test.Data, &test)
		if err != nil {
			return nil, nil, err
		}
		migratedTest, _ := expressions.MigrateTemplate(test.Test, nil)

		var delta int
		match := relativeDateTest.FindStringSubmatch(test.Test)
		if match != nil {
			delta, _ = strconv.Atoi(match[2])
			if match[1] == "-" {
				delta = -delta
			}
		}

		arguments = []string{migratedTest}

		caseUI = map[string]interface{}{
			"arguments": []string{strconv.Itoa(delta)},
		}

	// tests against a single group value
	case "in_group":
		test := groupTest{}
		err = json.Unmarshal(r.Test.Data, &test)
		arguments = []string{string(test.Test.UUID), string(test.Test.Name)}

	case "subflow":
		newType = "has_only_text"
		test := subflowTest{}
		err = json.Unmarshal(r.Test.Data, &test)
		arguments = []string{test.ExitType}

	case "webhook_status":
		newType = "has_only_text"
		test := webhookTest{}
		err = json.Unmarshal(r.Test.Data, &test)
		if test.Status == "success" {
			arguments = []string{"Success"}
		} else {
			arguments = []string{"Failure"}
		}

	case "airtime_status":
		newType = "has_only_text"
		test := airtimeTest{}
		err = json.Unmarshal(r.Test.Data, &test)
		if test.ExitStatus == "success" {
			arguments = []string{"Success"}
		} else {
			arguments = []string{"Failure"}
		}

	case "district":
		test := stringTest{}
		err = json.Unmarshal(r.Test.Data, &test)
		if err != nil {
			return nil, nil, err
		}

		migratedState, _ := expressions.MigrateTemplate(test.Test, nil)

		arguments = []string{migratedState}

	case "ward":
		test := wardTest{}
		err = json.Unmarshal(r.Test.Data, &test)
		if err != nil {
			return nil, nil, err
		}

		migratedDistrict, _ := expressions.MigrateTemplate(test.District, nil)
		migratedState, _ := expressions.MigrateTemplate(test.State, nil)

		arguments = []string{migratedDistrict, migratedState}

	default:
		return nil, nil, errors.Errorf("migration of '%s' tests not supported", r.Test.Type)
	}

	return routers.NewCase(caseUUID, newType, arguments, category.UUID()), caseUI, err
}

// migrates the given legacy actionset to a node with a set of migrated actions and a single exit
func migrateActionSet(lang utils.Language, a ActionSet, validDests map[flows.NodeUUID]bool, localization flows.Localization, baseMediaURL string) (flows.Node, error) {
	actions := make([]flows.Action, len(a.Actions))

	// migrate each action
	for i := range a.Actions {
		action, err := migrateAction(lang, a.Actions[i], localization, baseMediaURL)
		if err != nil {
			return nil, errors.Wrapf(err, "error migrating action[type=%s]", a.Actions[i].Type)
		}
		actions[i] = action
	}

	// only set exit destination if it's valid
	var destinationUUID flows.NodeUUID
	if validDests[a.Destination] {
		destinationUUID = a.Destination
	}

	exit := definition.NewExit(a.ExitUUID, destinationUUID)

	return definition.NewNode(a.UUID, actions, nil, []flows.Exit{exit}), nil
}

// ReadLegacyFlow reads a single legacy formatted flow
func ReadLegacyFlow(data json.RawMessage) (*Flow, error) {
	f := &Flow{}
	if err := utils.UnmarshalAndValidate(data, f); err != nil {
		return nil, err
	}

	if f.Metadata == nil {
		f.Metadata = &Metadata{}
	}

	return f, nil
}

func migrateNodes(f *Flow, baseMediaURL string) ([]flows.Node, map[flows.NodeUUID]*NodeUI, flows.Localization, error) {
	localization := definition.NewLocalization()
	numNodes := len(f.ActionSets) + len(f.RuleSets)
	nodes := make([]flows.Node, numNodes)
	nodeUIs := make(map[flows.NodeUUID]*NodeUI, numNodes)

	// get set of all node UUIDs, i.e. the valid destinations for any exit
	validDestinations := make(map[flows.NodeUUID]bool, numNodes)
	for _, as := range f.ActionSets {
		validDestinations[as.UUID] = true
	}
	for _, rs := range f.RuleSets {
		validDestinations[rs.UUID] = true
	}

	for i, actionSet := range f.ActionSets {
		node, err := migrateActionSet(f.BaseLanguage, actionSet, validDestinations, localization, baseMediaURL)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "error migrating action_set[uuid=%s]", actionSet.UUID)
		}
		nodes[i] = node
		nodeUIs[node.UUID()] = NewNodeUI(UINodeTypeActionSet, actionSet.X, actionSet.Y, nil)
	}

	for i, ruleSet := range f.RuleSets {
		node, uiType, uiNodeConfig, err := migrateRuleSet(f.BaseLanguage, ruleSet, validDestinations, localization)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "error migrating rule_set[uuid=%s]", ruleSet.UUID)
		}
		nodes[len(f.ActionSets)+i] = node
		nodeUIs[node.UUID()] = NewNodeUI(uiType, ruleSet.X, ruleSet.Y, uiNodeConfig)
	}

	// make sure our entry node is first
	var entryNodes, otherNodes []flows.Node
	for _, node := range nodes {
		if node.UUID() == f.Entry {
			entryNodes = []flows.Node{node}
		} else {
			otherNodes = append(otherNodes, node)
		}
	}

	// and sort remaining nodes by their top position (Y)
	sort.SliceStable(otherNodes, func(i, j int) bool {
		u1 := nodeUIs[otherNodes[i].UUID()]
		u2 := nodeUIs[otherNodes[j].UUID()]

		if u1 != nil && u2 != nil {
			return u1.Position.Top < u2.Position.Top
		}
		return false
	})

	nodes = append(entryNodes, otherNodes...)

	return nodes, nodeUIs, localization, nil
}

// Migrate migrates this legacy flow to the new format
func (f *Flow) Migrate(baseMediaURL string) (flows.Flow, error) {
	nodes, nodeUIs, localization, err := migrateNodes(f, baseMediaURL)
	if err != nil {
		return nil, err
	}

	// build UI section
	ui := NewUI()
	for _, actionSet := range f.ActionSets {
		ui.AddNode(actionSet.UUID, nodeUIs[actionSet.UUID])
	}
	for _, ruleSet := range f.RuleSets {
		ui.AddNode(ruleSet.UUID, nodeUIs[ruleSet.UUID])
	}
	for _, note := range f.Metadata.Notes {
		ui.AddSticky(note.Migrate())
	}

	uiJSON, err := json.Marshal(ui)
	if err != nil {
		return nil, err
	}

	uuid := f.Metadata.UUID
	name := f.Metadata.Name

	// some flows have these set on root-level instead.. or not set at all
	if uuid == "" {
		uuid = f.UUID
		if uuid == "" {
			uuid = assets.FlowUUID(utils.NewUUID())
		}
	}
	if name == "" {
		name = f.Name
	}

	return definition.NewFlow(
		uuid,
		name,
		f.BaseLanguage,
		flowTypeMapping[f.FlowType],
		f.Metadata.Revision,
		f.Metadata.Expires,
		localization,
		nodes,
		uiJSON,
	)
}
