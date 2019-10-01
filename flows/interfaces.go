package flows

import (
	"encoding/json"
	"time"

	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/excellent/types"
	"github.com/greatnonprofits-nfp/goflow/utils"
)

// NodeUUID is a UUID of a flow node
type NodeUUID utils.UUID

// CategoryUUID is the UUID of a node category
type CategoryUUID utils.UUID

// ExitUUID is the UUID of a node exit
type ExitUUID utils.UUID

// ActionUUID is the UUID of an action
type ActionUUID utils.UUID

// ContactID is the ID of a contact
type ContactID int64

// ContactUUID is the UUID of a contact
type ContactUUID utils.UUID

// RunUUID is the UUID of a flow run
type RunUUID utils.UUID

// StepUUID is the UUID of a run step
type StepUUID utils.UUID

// InputUUID is the UUID of an input
type InputUUID utils.UUID

// MsgID is the ID of a message
type MsgID int64

// NilMsgID is our constant for nil message ids
const NilMsgID = MsgID(0)

// MsgUUID is the UUID of a message
type MsgUUID utils.UUID

// FlowType represents the different types of flows
type FlowType string

const (
	// FlowTypeMessaging is a flow that is run over a messaging channel
	FlowTypeMessaging FlowType = "messaging"

	// FlowTypeMessagingOffline is a flow which is run over an offline messaging client like Surveyor
	FlowTypeMessagingOffline FlowType = "messaging_offline"

	// FlowTypeVoice is a flow which is run over IVR
	FlowTypeVoice FlowType = "voice"
)

// SessionStatus represents the current status of the engine session
type SessionStatus string

const (
	// SessionStatusActive represents a session that is still active
	SessionStatusActive SessionStatus = "active"

	// SessionStatusCompleted represents a session that has run to completion
	SessionStatusCompleted SessionStatus = "completed"

	// SessionStatusWaiting represents a session which is waiting for something from the caller
	SessionStatusWaiting SessionStatus = "waiting"

	// SessionStatusErrored represents a session that encountered an error
	SessionStatusErrored SessionStatus = "errored"
)

// RunStatus represents the current status of the flow run
type RunStatus string

const (
	// RunStatusActive represents a run that is still active
	RunStatusActive RunStatus = "active"

	// RunStatusCompleted represents a run that has run to completion
	RunStatusCompleted RunStatus = "completed"

	// RunStatusWaiting represents a run which is waiting for something from the caller
	RunStatusWaiting RunStatus = "waiting"

	// RunStatusErrored represents a run that encountered an error
	RunStatusErrored RunStatus = "errored"

	// RunStatusExpired represents a run that expired due to inactivity
	RunStatusExpired RunStatus = "expired"

	// RunStatusInterrupted represents a run that was interrupted by another flow
	RunStatusInterrupted RunStatus = "interrupted"
)

type FlowAssets interface {
	Get(assets.FlowUUID) (Flow, error)
}

// SessionAssets is the assets available to a session
type SessionAssets interface {
	Channels() *ChannelAssets
	Fields() *FieldAssets
	Flows() FlowAssets
	Groups() *GroupAssets
	Labels() *LabelAssets
	Locations() *LocationAssets
	Resthooks() *ResthookAssets
	Templates() *TemplateAssets
}

// Localizable is anything in the flow definition which can be localized and therefore needs a UUID
type Localizable interface {
	LocalizationUUID() utils.UUID
}

// Flow describes the ordered logic of actions and routers. It renders as its name in a template, and has the following
// properties which can be accessed:
//
//  * `uuid` the UUID of the flow
//  * `name` the name of the flow
//  * `revision` the revision number of the flow
//
// Examples:
//
//   @run.flow -> Registration
//   @child.run.flow.name -> Collect Age
//   @run.flow.uuid -> 50c3706e-fedb-42c0-8eab-dda3335714b7
//   @(json(run.flow)) -> {"name":"Registration","revision":123,"uuid":"50c3706e-fedb-42c0-8eab-dda3335714b7"}
//
// @context flow
type Flow interface {
	Contextable

	// spec properties
	UUID() assets.FlowUUID
	Name() string
	Revision() int
	Language() utils.Language
	Type() FlowType
	ExpireAfterMinutes() int
	Localization() Localization
	UI() json.RawMessage
	Nodes() []Node
	GetNode(uuid NodeUUID) Node
	Reference() *assets.FlowReference
	Generic() map[string]interface{}
	Clone(map[utils.UUID]utils.UUID) Flow

	Inspect() *FlowInfo
	Validate(SessionAssets, func(assets.Reference)) error
	ValidateRecursive(SessionAssets, func(assets.Reference)) error

	ExtractTemplates() []string
	RewriteTemplates(func(string) string)
	ExtractDependencies() []assets.Reference
	ExtractResults() []*ResultInfo

	MarshalWithInfo() ([]byte, error)
}

// Node is a single node in a flow
type Node interface {
	Inspectable

	UUID() NodeUUID
	Actions() []Action
	Router() Router
	Exits() []Exit

	Validate(Flow, map[utils.UUID]bool) error
}

// Action is an action within a flow node
type Action interface {
	utils.Typed
	Localizable
	Inspectable

	UUID() ActionUUID
	Execute(FlowRun, Step, ModifierCallback, EventCallback) error
	Validate() error
	AllowedFlowTypes() []FlowType
}

type Router interface {
	utils.Typed
	Inspectable

	Wait() Wait
	ResultName() string

	Validate([]Exit) error
	AllowTimeout() bool
	Route(FlowRun, Step, EventCallback) (ExitUUID, error)
	RouteTimeout(FlowRun, Step, EventCallback) (ExitUUID, error)
}

type Exit interface {
	UUID() ExitUUID
	DestinationUUID() NodeUUID
}

type Timeout interface {
	Seconds() int
	CategoryUUID() CategoryUUID
}

type Wait interface {
	utils.Typed

	Timeout() Timeout

	Begin(FlowRun, EventCallback) ActivatedWait
	End(Resume) error
}

type ActivatedWait interface {
	utils.Typed

	TimeoutSeconds() *int
}

type Hint interface {
	utils.Typed
}

// Localization provide a way to get the translations for a specific language
type Localization interface {
	AddItemTranslation(utils.Language, utils.UUID, string, []string)
	GetTranslations(utils.Language) Translations
	Languages() []utils.Language
}

// Translations provide a way to get the translation for a specific language for a uuid/key pair
type Translations interface {
	GetTextArray(utils.UUID, string) []string
	SetTextArray(utils.UUID, string, []string)
}

// Trigger represents something which can initiate a session with the flow engine. It has several properties which can be
// accessed in expressions:
//
//  * `type` the type of the trigger, one of "manual" or "flow"
//  * `params` the parameters passed to the trigger
//
// Examples:
//
//   @trigger.type -> flow_action
//   @trigger.params -> {address: {state: WA}, source: website}
//   @(json(trigger)) -> {"params":{"address":{"state":"WA"},"source":"website"},"type":"flow_action"}
//
// @context trigger
type Trigger interface {
	utils.Typed
	Contextable

	Initialize(Session, EventCallback) error
	InitializeRun(FlowRun, EventCallback) error

	Environment() utils.Environment
	Flow() *assets.FlowReference
	Contact() *Contact
	Connection() *Connection
	Params() types.XValue
	TriggeredOn() time.Time
}

// TriggerWithRun is special case of trigger that provides a parent run to the session
type TriggerWithRun interface {
	Trigger

	RunSummary() json.RawMessage
}

// Resume represents something which can resume a session with the flow engine
type Resume interface {
	utils.Typed

	Apply(FlowRun, EventCallback) error

	Environment() utils.Environment
	Contact() *Contact
	ResumedOn() time.Time
}

// Modifier is something which can modify a contact
type Modifier interface {
	utils.Typed

	Apply(utils.Environment, SessionAssets, *Contact, EventCallback)
}

// ModifierCallback is a callback invoked when a modifier has been generated
type ModifierCallback func(Modifier)

// Event describes a state change
type Event interface {
	utils.Typed

	CreatedOn() time.Time
	StepUUID() StepUUID
	SetStepUUID(StepUUID)
}

// EventCallback is a callback invoked when an event has been generated
type EventCallback func(Event)

// Input describes input from the contact and currently we only support one type of input: `msg`. Any input has the following
// properties which can be accessed:
//
//  * `uuid` the UUID of the input
//  * `type` the type of the input, e.g. `msg`
//  * `channel` the [channel](#context:channel) that the input was received on
//  * `created_on` the time when the input was created
//
// An input of type `msg` renders as its text and attachments in a template, and has the following additional properties:
//
//  * `text` the text of the message
//  * `attachments` any [attachments](#context:attachment) on the message
//  * `urn` the [URN](#context:urn) that the input was received on
//
// Examples:
//
//   @input -> Hi there\nhttp://s3.amazon.com/bucket/test.jpg\nhttp://s3.amazon.com/bucket/test.mp3
//   @input.type -> msg
//   @input.text -> Hi there
//   @input.attachments -> [image/jpeg:http://s3.amazon.com/bucket/test.jpg, audio/mp3:http://s3.amazon.com/bucket/test.mp3]
//   @(json(input)) -> {"attachments":["image/jpeg:http://s3.amazon.com/bucket/test.jpg","audio/mp3:http://s3.amazon.com/bucket/test.mp3"],"channel":{"address":"+12345671111","name":"My Android Phone","uuid":"57f1078f-88aa-46f4-a59a-948a5739c03d"},"created_on":"2017-12-31T11:35:10.035757-02:00","text":"Hi there","type":"msg","urn":"tel:+12065551212","uuid":"9bf91c2b-ce58-4cef-aacc-281e03f69ab5"}
//
// @context input
type Input interface {
	utils.Typed
	Contextable

	UUID() InputUUID
	CreatedOn() time.Time
	Channel() *Channel
}

type Step interface {
	Contextable

	UUID() StepUUID
	NodeUUID() NodeUUID
	ExitUUID() ExitUUID
	ArrivedOn() time.Time

	Leave(ExitUUID)
}

type Engine interface {
	NewSession(SessionAssets, Trigger) (Session, Sprint, error)
	ReadSession(SessionAssets, json.RawMessage, assets.MissingCallback) (Session, error)

	HTTPClient() *utils.HTTPClient
	DisableWebhooks() bool
	MaxWebhookResponseBytes() int
	MaxStepsPerSprint() int
}

// Sprint is an interaction with the engine - i.e. a start or resume of a session
type Sprint interface {
	Modifiers() []Modifier
	LogModifier(Modifier)
	Events() []Event
	LogEvent(Event)
}

// Session represents the session of a flow run which may contain many runs
type Session interface {
	Assets() SessionAssets

	Type() FlowType
	SetType(FlowType)

	Environment() utils.Environment
	SetEnvironment(utils.Environment)

	Contact() *Contact
	SetContact(*Contact)

	Input() Input
	SetInput(Input)

	Status() SessionStatus
	Trigger() Trigger
	PushFlow(Flow, FlowRun, bool)
	Wait() ActivatedWait

	Resume(Resume) (Sprint, error)
	Runs() []FlowRun
	GetRun(RunUUID) (FlowRun, error)
	GetCurrentChild(FlowRun) FlowRun
	ParentRun() RunSummary

	Engine() Engine
}

// RunSummary represents the minimum information available about all runs (current or related) and is the
// representation of runs made accessible to router tests.
type RunSummary interface {
	UUID() RunUUID
	Contact() *Contact
	Flow() Flow
	Status() RunStatus
	Results() Results
}

// RunEnvironment is a run specific environment which adds location functionality required by some router tests
type RunEnvironment interface {
	utils.Environment

	FindLocations(string, utils.LocationLevel, *utils.Location) ([]*utils.Location, error)
	FindLocationsFuzzy(string, utils.LocationLevel, *utils.Location) ([]*utils.Location, error)
	LookupLocation(utils.LocationPath) (*utils.Location, error)
}

// FlowRun is a single contact's journey through a flow. It records the path they have taken, and the results that have been
// collected. It has several properties which can be accessed in expressions:
//
//  * `uuid` the UUID of the run
//  * `flow` the [flow](#context:flow) of the run
//  * `contact` the [contact](#context:contact) of the flow run
//  * `input` the [input](#context:input) of the current run
//  * `results` the results that have been saved for this run
//  * `results.[snaked_result_name]` the value of the specific result, e.g. `results.age`
//
// Examples:
//
//   @run -> Ryan Lewis@Registration
//   @run.contact.name -> Ryan Lewis
//   @run.flow.name -> Registration
//
// @context run
type FlowRun interface {
	Contextable
	RunSummary

	Environment() RunEnvironment
	Session() Session
	SaveResult(*Result)
	SetStatus(RunStatus)

	CreateStep(Node) Step
	Path() []Step
	PathLocation() (Step, Node, error)

	LogEvent(Step, Event)
	LogError(Step, error)
	Events() []Event

	EvaluateTemplateValue(template string) (types.XValue, error)
	EvaluateTemplate(template string) (string, error)

	GetText(utils.UUID, string, string) string
	GetTextArray(utils.UUID, string, []string) []string
	GetTranslatedTextArray(utils.UUID, string, []string, []utils.Language) []string

	Snapshot() RunSummary
	Parent() RunSummary
	ParentInSession() FlowRun
	Ancestors() []FlowRun

	CreatedOn() time.Time
	ModifiedOn() time.Time
	ExpiresOn() *time.Time
	ResetExpiration(*time.Time)
	ExitedOn() *time.Time
	Exit(RunStatus)
}

// LegacyExtraContributor is something which contributes results for constructing @legacy_extra
type LegacyExtraContributor interface {
	LegacyExtra() Results
}
