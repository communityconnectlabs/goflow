package engine

import (
	"encoding/json"

	"github.com/nyaruka/gocommon/uuids"
	"github.com/nyaruka/goflow/assets"
	"github.com/nyaruka/goflow/envs"
	"github.com/nyaruka/goflow/excellent"
	"github.com/nyaruka/goflow/excellent/types"
	"github.com/nyaruka/goflow/flows"
)

// an instance of the engine
type engine struct {
	evaluator       *excellent.Evaluator
	services        *services
	options         *flows.EngineOptions
	warningCallback func(string)
}

// NewSession creates a new session
func (e *engine) NewSession(sa flows.SessionAssets, trigger flows.Trigger) (flows.Session, flows.Sprint, error) {
	s := &session{
		uuid:       flows.SessionUUID(uuids.New()),
		env:        envs.NewBuilder().Build(),
		engine:     e,
		assets:     sa,
		trigger:    trigger,
		status:     flows.SessionStatusActive,
		batchStart: trigger.Batch(),
		runsByUUID: make(map[flows.RunUUID]flows.Run),
	}

	sprint, err := s.start(trigger)

	return s, sprint, err
}

// ReadSession reads an existing session
func (e *engine) ReadSession(sa flows.SessionAssets, data json.RawMessage, missing assets.MissingCallback) (flows.Session, error) {
	return readSession(e, sa, data, missing)
}

func (e *engine) Evaluator() *excellent.Evaluator { return e.evaluator }
func (e *engine) Services() flows.Services        { return e.services }
func (e *engine) Options() *flows.EngineOptions   { return e.options }

func (e *engine) onDeprecatedContextValue(v types.XValue) {
	if e.warningCallback != nil {
		e.warningCallback("deprecated context access: " + v.Deprecated())
	}
}

var _ flows.Engine = (*engine)(nil)

//------------------------------------------------------------------------------------------
// Builder
//------------------------------------------------------------------------------------------

// Builder is a builder for engine configs
type Builder struct {
	eng *engine
}

// NewBuilder creates a new engine builder
func NewBuilder() *Builder {
	e := &engine{
		services: newEmptyServices(),
		options: &flows.EngineOptions{
			MaxStepsPerSprint:    100,
			MaxResumesPerSession: 500,
			MaxTemplateChars:     10000,
			MaxFieldChars:        640,
			MaxResultChars:       640,
		},
	}
	e.evaluator = excellent.NewEvaluator(excellent.WithDeprecatedCallback(e.onDeprecatedContextValue))
	return &Builder{eng: e}
}

// WithEmailServiceFactory sets the email service factory
func (b *Builder) WithEmailServiceFactory(f EmailServiceFactory) *Builder {
	b.eng.services.email = f
	return b
}

// WithWebhookServiceFactory sets the webhook service factory
func (b *Builder) WithWebhookServiceFactory(f WebhookServiceFactory) *Builder {
	b.eng.services.webhook = f
	return b
}

// WithClassificationServiceFactory sets the NLU service factory
func (b *Builder) WithClassificationServiceFactory(f ClassificationServiceFactory) *Builder {
	b.eng.services.classification = f
	return b
}

// WithAirtimeServiceFactory sets the airtime service factory
func (b *Builder) WithAirtimeServiceFactory(f AirtimeServiceFactory) *Builder {
	b.eng.services.airtime = f
	return b
}

// WithMaxStepsPerSprint sets the maximum number of steps allowed in a single sprint
func (b *Builder) WithMaxStepsPerSprint(max int) *Builder {
	b.eng.options.MaxStepsPerSprint = max
	return b
}

// WithMaxResumesPerSession sets the maximum number of resumes allowed in a single session
func (b *Builder) WithMaxResumesPerSession(max int) *Builder {
	b.eng.options.MaxResumesPerSession = max
	return b
}

// WithMaxTemplateChars sets the maximum number of characters allowed from an evaluated template
func (b *Builder) WithMaxTemplateChars(max int) *Builder {
	b.eng.options.MaxTemplateChars = max
	return b
}

// WithMaxFieldChars sets the maximum number of characters allowed in a contact field value
func (b *Builder) WithMaxFieldChars(max int) *Builder {
	b.eng.options.MaxFieldChars = max
	return b
}

// WithMaxResultChars sets the maximum number of characters allowed in a result value
func (b *Builder) WithMaxResultChars(max int) *Builder {
	b.eng.options.MaxResultChars = max
	return b
}

// WithWarningCallback sets the email service factory
func (b *Builder) WithWarningCallback(callback func(string)) *Builder {
	b.eng.warningCallback = callback
	return b
}

// Build returns the final engine
func (b *Builder) Build() flows.Engine { return b.eng }
