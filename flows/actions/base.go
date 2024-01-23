package actions

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/urns"
	"github.com/nyaruka/gocommon/uuids"
	"github.com/nyaruka/goflow/assets"
	"github.com/nyaruka/goflow/envs"
	"github.com/nyaruka/goflow/excellent/types"
	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/flows/events"
	"github.com/nyaruka/goflow/utils"

	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
)

// max number of bytes to be saved to extra on a result
const resultExtraMaxBytes = 10000

// max length of a message attachment (type:url)
const maxAttachmentLength = 2048

// common category names
const (
	CategorySuccess  = "Success"
	CategorySkipped  = "Skipped"
	CategoryFailure  = "Failure"
	CategoryAnswer   = "Answer"
	CategoryNoAnswer = "No Answer"
)

var webhookCategories = []string{CategorySuccess, CategoryFailure}
var webhookStatusCategories = map[flows.CallStatus]string{
	flows.CallStatusSuccess:         CategorySuccess,
	flows.CallStatusResponseError:   CategoryFailure,
	flows.CallStatusConnectionError: CategoryFailure,
	flows.CallStatusSubscriberGone:  CategoryFailure,
}

var voiceCallCategories = []string{CategoryAnswer, CategoryNoAnswer, CategoryFailure}
var voiceCallStatusCategories = map[flows.CallStatus]string{
	flows.CallStatusVoiceHuman:        CategoryAnswer,
	flows.CallStatusVoiceUnknown:      CategoryAnswer,
	flows.CallStatusMachineEndBeep:    CategoryNoAnswer,
	flows.CallStatusMachineEndSilence: CategoryNoAnswer,
	flows.CallStatusMachineEndOther:   CategoryNoAnswer,
	flows.CallStatusResponseError:     CategoryFailure,
	flows.CallStatusConnectionError:   CategoryFailure,
}

var registeredTypes = map[string](func() flows.Action){}

// registers a new type of action
func registerType(name string, initFunc func() flows.Action) {
	registeredTypes[name] = initFunc
}

// RegisteredTypes gets the registered types of action
func RegisteredTypes() map[string](func() flows.Action) {
	return registeredTypes
}

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// the base of all action types
type baseAction struct {
	Type_ string           `json:"type" validate:"required"`
	UUID_ flows.ActionUUID `json:"uuid" validate:"required,uuid4"`
}

// creates a new base action
func newBaseAction(typeName string, uuid flows.ActionUUID) baseAction {
	return baseAction{Type_: typeName, UUID_: uuid}
}

// Type returns the type of this action
func (a *baseAction) Type() string { return a.Type_ }

// UUID returns the UUID of the action
func (a *baseAction) UUID() flows.ActionUUID { return a.UUID_ }

// Validate validates our action is valid
func (a *baseAction) Validate() error { return nil }

// LocalizationUUID gets the UUID which identifies this object for localization
func (a *baseAction) LocalizationUUID() uuids.UUID { return uuids.UUID(a.UUID_) }

// helper function for actions that send a message (text + attachments) that must be localized and evalulated
func (a *baseAction) evaluateMessage(run flows.Run, languages []envs.Language, actionText string, actionAttachments []string, actionQuickReplies []string, logEvent flows.EventCallback) (string, []utils.Attachment, []string) {
	// localize and evaluate the message text
	localizedText := run.GetTranslatedTextArray(uuids.UUID(a.UUID()), "text", []string{actionText}, languages)[0]
	evaluatedText, err := run.EvaluateTemplate(localizedText)
	if err != nil {
		logEvent(events.NewError(err))
	}

	// localize and evaluate the message attachments
	translatedAttachments := run.GetTranslatedTextArray(uuids.UUID(a.UUID()), "attachments", actionAttachments, languages)
	evaluatedAttachments := make([]utils.Attachment, 0, len(translatedAttachments))
	for _, a := range translatedAttachments {
		evaluatedAttachment, err := run.EvaluateTemplate(a)
		if err != nil {
			logEvent(events.NewError(err))
		}
		if evaluatedAttachment == "" {
			logEvent(events.NewErrorf("attachment text evaluated to empty string, skipping"))
			continue
		}
		if len(evaluatedAttachment) > maxAttachmentLength {
			logEvent(events.NewErrorf("evaluated attachment is longer than %d limit, skipping", maxAttachmentLength))
			continue
		}
		evaluatedAttachments = append(evaluatedAttachments, utils.Attachment(evaluatedAttachment))
	}

	// localize and evaluate the quick replies
	translatedQuickReplies := run.GetTranslatedTextArray(uuids.UUID(a.UUID()), "quick_replies", actionQuickReplies, languages)
	evaluatedQuickReplies := make([]string, 0, len(translatedQuickReplies))
	for _, qr := range translatedQuickReplies {
		evaluatedQuickReply, err := run.EvaluateTemplate(qr)
		if err != nil {
			logEvent(events.NewError(err))
		}
		if evaluatedQuickReply == "" {
			logEvent(events.NewErrorf("quick reply text evaluated to empty string, skipping"))
			continue
		}
		evaluatedQuickReplies = append(evaluatedQuickReplies, evaluatedQuickReply)
	}

	return evaluatedText, evaluatedAttachments, evaluatedQuickReplies
}

// helper to save a run result and log it as an event
func (a *baseAction) saveResult(run flows.Run, step flows.Step, name, value, category, categoryLocalized string, input string, extra json.RawMessage, logEvent flows.EventCallback) {
	result := flows.NewResult(name, value, category, categoryLocalized, step.NodeUUID(), input, extra, dates.Now(), "")
	run.SaveResult(result)
	logEvent(events.NewRunResultChanged(result))
}

// helper to save a run result based on a webhook call and log it as an event
func (a *baseAction) saveWebhookResult(run flows.Run, step flows.Step, name string, call *flows.WebhookCall, status flows.CallStatus, logEvent flows.EventCallback) {
	input := fmt.Sprintf("%s %s", call.Request.Method, call.Request.URL.String())
	value := "0"
	category := webhookStatusCategories[status]
	var extra json.RawMessage

	if call.Response != nil {
		value = strconv.Itoa(call.Response.StatusCode)

		if len(call.ResponseJSON) > 0 && len(call.ResponseJSON) < resultExtraMaxBytes {
			extra = call.ResponseJSON
		}
	}

	a.saveResult(run, step, name, value, category, "", input, extra, logEvent)
}

// helper to save a run result based on a voice call and log it as an event
func (a *baseAction) saveVoiceCallResult(run flows.Run, step flows.Step, name string, call *flows.WebhookCall, status flows.CallStatus, logEvent flows.EventCallback) {
	value := "0"
	category := voiceCallStatusCategories[status]
	var extra json.RawMessage

	if call.Response != nil {
		value = strconv.Itoa(call.Response.StatusCode)
	}

	a.saveResult(run, step, name, value, category, "", "", extra, logEvent)
}

func (a *baseAction) updateWebhook(run flows.Run, call *flows.WebhookCall) {
	parsed := types.JSONToXValue(call.ResponseJSON)

	switch typed := parsed.(type) {
	case nil, types.XError:
		run.SetWebhook(types.XObjectEmpty)
	default:
		run.SetWebhook(typed)
	}
}

// helper to apply a contact modifier
func (a *baseAction) applyModifier(run flows.Run, mod flows.Modifier, logModifier flows.ModifierCallback, logEvent flows.EventCallback) {
	mod.Apply(run.Environment(), run.Session().Assets(), run.Contact(), logEvent)
	logModifier(mod)
}

// helper to log a failure
func (a *baseAction) fail(run flows.Run, err error, logEvent flows.EventCallback) {
	run.Exit(flows.RunStatusFailed)
	logEvent(events.NewFailure(err))
}

// utility struct which sets the allowed flow types to any
type universalAction struct{}

// AllowedFlowTypes returns the flow types which this action is allowed to occur in
func (a *universalAction) AllowedFlowTypes() []flows.FlowType {
	return []flows.FlowType{flows.FlowTypeMessaging, flows.FlowTypeMessagingBackground, flows.FlowTypeMessagingOffline, flows.FlowTypeVoice}
}

// utility struct which sets the allowed flow types to non-background
type interactiveAction struct{}

// AllowedFlowTypes returns the flow types which this action is allowed to occur in
func (a *interactiveAction) AllowedFlowTypes() []flows.FlowType {
	return []flows.FlowType{flows.FlowTypeMessaging, flows.FlowTypeMessagingOffline, flows.FlowTypeVoice}
}

// utility struct which sets the allowed flow types to any which run online
type onlineAction struct{}

// AllowedFlowTypes returns the flow types which this action is allowed to occur in
func (a *onlineAction) AllowedFlowTypes() []flows.FlowType {
	return []flows.FlowType{flows.FlowTypeMessaging, flows.FlowTypeMessagingBackground, flows.FlowTypeVoice}
}

// utility struct which sets the allowed flow types to just voice
type voiceAction struct{}

// AllowedFlowTypes returns the flow types which this action is allowed to occur in
func (a *voiceAction) AllowedFlowTypes() []flows.FlowType {
	return []flows.FlowType{flows.FlowTypeVoice}
}

// utility struct for actions which operate on other contacts
type otherContactsAction struct {
	URNs         []urns.URN                `json:"urns,omitempty"`
	Groups       []*assets.GroupReference  `json:"groups,omitempty" validate:"dive"`
	Contacts     []*flows.ContactReference `json:"contacts,omitempty" validate:"dive"`
	ContactQuery string                    `json:"contact_query,omitempty" engine:"evaluated"`
	LegacyVars   []string                  `json:"legacy_vars,omitempty" engine:"evaluated"`
}

func (a *otherContactsAction) resolveRecipients(run flows.Run, logEvent flows.EventCallback) ([]*assets.GroupReference, []*flows.ContactReference, string, []urns.URN, error) {
	groupSet := run.Session().Assets().Groups()

	// copy URNs
	urnList := make([]urns.URN, 0, len(a.URNs))
	urnList = append(urnList, a.URNs...)

	// copy contact references
	contactRefs := make([]*flows.ContactReference, 0, len(a.Contacts))
	contactRefs = append(contactRefs, a.Contacts...)

	// resolve group references
	groups := resolveGroups(run, a.Groups, logEvent)
	groupRefs := make([]*assets.GroupReference, 0, len(groups))
	for _, group := range groups {
		groupRefs = append(groupRefs, group.Reference())
	}

	// evaluate the legacy variables
	for _, legacyVar := range a.LegacyVars {
		evaluatedLegacyVar, err := run.EvaluateTemplate(legacyVar)
		if err != nil {
			logEvent(events.NewError(err))
		}

		evaluatedLegacyVar = strings.TrimSpace(evaluatedLegacyVar)

		if uuidRegex.MatchString(evaluatedLegacyVar) {
			// if variable evaluates to a UUID, we assume it's a contact UUID
			contactRefs = append(contactRefs, flows.NewContactReference(flows.ContactUUID(evaluatedLegacyVar), ""))

		} else if groupByName := groupSet.FindByName(evaluatedLegacyVar); groupByName != nil {
			// next up we look for a group with a matching name
			groupRefs = append(groupRefs, groupByName.Reference())
		} else {
			// next up try it as a URN
			urn := urns.URN(evaluatedLegacyVar)
			if urn.Validate() == nil {
				urn = urn.Normalize(string(run.Environment().DefaultCountry()))
				urnList = append(urnList, urn)
			} else {
				// if that fails, try to parse as phone number
				parsedTel := utils.ParsePhoneNumber(evaluatedLegacyVar, string(run.Environment().DefaultCountry()))
				if parsedTel != "" {
					urn, _ := urns.NewURNFromParts(urns.TelScheme, parsedTel, "", "")
					urnList = append(urnList, urn)
				} else {
					logEvent(events.NewErrorf("'%s' couldn't be resolved to a contact, group or URN", evaluatedLegacyVar))
				}
			}
		}
	}

	// evaluate contact query
	contactQuery, _ := run.EvaluateTemplateText(a.ContactQuery, flows.ContactQueryEscaping, true)
	contactQuery = strings.TrimSpace(contactQuery)

	return groupRefs, contactRefs, contactQuery, urnList, nil
}

// utility struct for actions which create a message
type createMsgAction struct {
	Text         string   `json:"text" validate:"required" engine:"localized,evaluated"`
	Attachments  []string `json:"attachments,omitempty" engine:"localized,evaluated"`
	QuickReplies []string `json:"quick_replies,omitempty" engine:"localized,evaluated"`
}

// helper function for actions that have a set of group references that must be resolved to actual groups
func resolveGroups(run flows.Run, references []*assets.GroupReference, logEvent flows.EventCallback) []*flows.Group {
	groupAssets := run.Session().Assets().Groups()
	groups := make([]*flows.Group, 0, len(references))

	for _, ref := range references {
		var group *flows.Group

		if ref.Variable() {
			// is an expression that evaluates to an existing group's name
			evaluatedName, err := run.EvaluateTemplate(ref.NameMatch)
			if err != nil {
				logEvent(events.NewError(err))
			} else {
				// look up the set of all groups to see if such a group exists
				group = groupAssets.FindByName(evaluatedName)
				if group == nil {
					logEvent(events.NewErrorf("no such group with name '%s'", evaluatedName))
				}
			}
		} else {
			// group is a fixed group with a UUID
			group = groupAssets.Get(ref.UUID)
			if group == nil {
				logEvent(events.NewDependencyError(ref))
			}
		}

		if group != nil {
			groups = append(groups, group)
		}
	}

	return groups
}

// helper function for actions that have a set of label references that must be resolved to actual labels
func resolveLabels(run flows.Run, references []*assets.LabelReference, logEvent flows.EventCallback) []*flows.Label {
	labelAssets := run.Session().Assets().Labels()
	labels := make([]*flows.Label, 0, len(references))

	for _, ref := range references {
		var label *flows.Label

		if ref.Variable() {
			// is an expression that evaluates to an existing label's name
			evaluatedName, err := run.EvaluateTemplate(ref.NameMatch)
			if err != nil {
				logEvent(events.NewError(err))
			} else {
				// look up the set of all labels to see if such a label exists
				label = labelAssets.FindByName(evaluatedName)
				if label == nil {
					logEvent(events.NewErrorf("no such label with name '%s'", evaluatedName))
				}
			}
		} else {
			// label is a fixed label with a UUID
			label = labelAssets.Get(ref.UUID)
			if label == nil {
				logEvent(events.NewDependencyError(ref))
			}
		}

		if label != nil {
			labels = append(labels, label)
		}
	}

	return labels
}

// helper function to resolve a user reference to a user
func resolveUser(run flows.Run, ref *assets.UserReference, logEvent flows.EventCallback) *flows.User {
	userAssets := run.Session().Assets().Users()
	var user *flows.User

	if ref.Variable() {
		// is an expression that evaluates to an existing user's email
		evaluatedEmail, err := run.EvaluateTemplate(ref.EmailMatch)
		if err != nil {
			logEvent(events.NewError(err))
		} else {
			// look up to see if such a user exists
			user = userAssets.Get(evaluatedEmail)
			if user == nil {
				logEvent(events.NewErrorf("no such user with email '%s'", evaluatedEmail))
			}
		}
	} else {
		// user is a fixed user with this email address
		user = userAssets.Get(ref.Email)
		if user == nil {
			logEvent(events.NewDependencyError(ref))
		}
	}

	return user
}

//------------------------------------------------------------------------------------------
// JSON Encoding / Decoding
//------------------------------------------------------------------------------------------

// ReadAction reads an action from the given JSON
func ReadAction(data json.RawMessage) (flows.Action, error) {
	typeName, err := utils.ReadTypeFromJSON(data)
	if err != nil {
		return nil, err
	}

	f := registeredTypes[typeName]
	if f == nil {
		return nil, errors.Errorf("unknown type: '%s'", typeName)
	}

	action := f()
	return action, utils.UnmarshalAndValidate(data, action)
}

func findDestinationInLinks(dest string, links []string) (string, string) {
	for _, link := range links {
		linkSplitted := strings.SplitN(link, ":", 2)
		destSplitted := strings.SplitN(dest, "?", 2)
		if destSplitted[0] == linkSplitted[1] {
			return linkSplitted[0], linkSplitted[1]
		}
	}
	return "", ""
}

func generateTextWithShortenLinks(text string, orgLinks []string, contactUUID string, flowUUID string, host string) string {
	URLshHost := utils.GetEnv(utils.URLshHost, "")
	if host != "" {
		URLshHost = host
	}
	URLshToken := utils.GetEnv(utils.URLshToken, "")
	mailroomDomain := utils.GetEnv(utils.MailroomDomain, "")

	generatedText := text

	// Whether we don't have the URLsh credentials, should be skipped
	if URLshHost == "" || URLshToken == "" || mailroomDomain == "" {
		return generatedText
	}

	// splitting the text as array for analyzing and replace if it's the case
	re := regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?!&//=]*)`)
	linksFound := re.FindAllString(text, -1)

	for _, d := range linksFound {
		// checking if the text is a valid URL
		if !isValidURL(d) {
			continue
		}

		destUUID, destLink := findDestinationInLinks(d, orgLinks)

		if destUUID == "" || destLink == "" {
			continue
		}

		if contactUUID != "" {
			urlshURL := fmt.Sprintf("%s/api/admin/urls", URLshHost)
			handleURL := fmt.Sprintf("https://%s/link/handler/%s", mailroomDomain, destUUID)
			longURL := fmt.Sprintf("%s?contact=%s&flow=%s&full_link=%s", handleURL, contactUUID, flowUUID, d)

			// build our request
			method := "POST"

			payload := map[string]string{
				"url": longURL,
			}
			payloadString, _ := json.Marshal(payload)

			req, errReq := http.NewRequest(method, urlshURL, strings.NewReader(string(payloadString)))
			if errReq != nil {
				continue
			}

			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", URLshToken))
			req.Header.Add("Content-Type", "application/json")

			resp, errHttp := http.DefaultClient.Do(req)
			if errHttp != nil {
				continue
			}
			content, errRead := ioutil.ReadAll(resp.Body)
			if errRead != nil {
				continue
			}

			// replacing the link for the YoURLs generated link
			shortLink, _ := jsonparser.GetString(content, "short_url")
			generatedText = strings.Replace(generatedText, d, shortLink, -1)
		}

	}

	return generatedText

}
