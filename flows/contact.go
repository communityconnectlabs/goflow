package flows

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/nyaruka/gocommon/urns"
	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/contactql"
	"github.com/greatnonprofits-nfp/goflow/excellent/types"
	"github.com/greatnonprofits-nfp/goflow/utils"

	"github.com/pkg/errors"
)

// Contact represents a person who is interacting with the flow. It renders as the person's name
// (or perferred URN if name isn't set) in a template, and has the following properties which can be accessed:
//
//  * `uuid` the UUID of the contact
//  * `name` the full name of the contact
//  * `first_name` the first name of the contact
//  * `language` the [ISO-639-3](http://www-01.sil.org/iso639-3/) language code of the contact
//  * `timezone` the timezone name of the contact
//  * `created_on` the datetime when the contact was created
//  * `urns` all [URNs](#context:urn) the contact has set
//  * `urns.[scheme]` all the [URNs](#context:urn) the contact has set for the particular URN scheme
//  * `urn` shorthand for `@(format_urn(c.urns.0))`, i.e. the contact's preferred [URN](#context:urn) in friendly formatting
//  * `groups` all the [groups](#context:group) that the contact belongs to
//  * `fields` all the custom contact fields the contact has set
//  * `fields.[snaked_field_name]` the value of the specific field
//  * `channel` shorthand for `contact.urns[0].channel`, i.e. the [channel](#context:channel) of the contact's preferred URN
//
// Examples:
//
//   @contact.name -> Ryan Lewis
//   @contact.first_name -> Ryan
//   @contact.language -> eng
//   @contact.timezone -> America/Guayaquil
//   @contact.created_on -> 2018-06-20T11:40:30.123456Z
//   @contact.urns -> [tel:+12065551212, twitterid:54784326227#nyaruka, mailto:foo@bar.com]
//   @(contact.urns[0]) -> tel:+12065551212
//   @contact.urn -> tel:+12065551212
//   @(foreach(contact.groups, extract, "name")) -> [Testers, Males]
//   @contact.fields -> Activation Token: AACC55\nAge: 23\nGender: Male\nJoin Date: 2017-12-02T00:00:00.000000-02:00
//   @contact.fields.activation_token -> AACC55
//   @contact.fields.gender -> Male
//
// @context contact
type Contact struct {
	uuid      ContactUUID
	id        ContactID
	name      string
	language  utils.Language
	timezone  *time.Location
	createdOn time.Time
	urns      URNList
	groups    *GroupList
	fields    FieldValues

	// transient fields
	assets SessionAssets
}

// NewContact creates a new contact with the passed in attributes
func NewContact(
	sa SessionAssets,
	uuid ContactUUID,
	id ContactID,
	name string,
	language utils.Language,
	timezone *time.Location,
	createdOn time.Time,
	urns []urns.URN,
	groups []assets.Group,
	fields map[string]*Value) (*Contact, error) {

	urnList, err := ReadURNList(sa, urns, assets.IgnoreMissing)
	if err != nil {
		return nil, err
	}

	groupList, err := NewGroupListFromAssets(sa, groups)
	if err != nil {
		return nil, err
	}

	fieldValues, err := NewFieldValues(sa, fields, assets.IgnoreMissing)
	if err != nil {
		return nil, err
	}

	return &Contact{
		uuid:      uuid,
		id:        id,
		name:      name,
		language:  language,
		timezone:  timezone,
		createdOn: createdOn,
		urns:      urnList,
		groups:    groupList,
		fields:    fieldValues,
		assets:    sa,
	}, nil
}

// NewEmptyContact creates a new empy contact with the passed in name, language and location
func NewEmptyContact(sa SessionAssets, name string, language utils.Language, timezone *time.Location) *Contact {
	return &Contact{
		uuid:      ContactUUID(utils.NewUUID()),
		name:      name,
		language:  language,
		timezone:  timezone,
		createdOn: utils.Now(),
		urns:      URNList{},
		groups:    NewGroupList([]*Group{}),
		fields:    make(FieldValues),
		assets:    sa,
	}
}

// Clone creates a copy of this contact
func (c *Contact) Clone() *Contact {
	if c == nil {
		return nil
	}

	return &Contact{
		uuid:      c.uuid,
		id:        c.id,
		name:      c.name,
		language:  c.language,
		timezone:  c.timezone,
		createdOn: c.createdOn,
		urns:      c.urns.clone(),
		groups:    c.groups.clone(),
		fields:    c.fields.clone(),
		assets:    c.assets,
	}
}

// Equal returns true if this instance is equal to the given instance
func (c *Contact) Equal(other *Contact) bool {
	asJSON1, _ := json.Marshal(c)
	asJSON2, _ := json.Marshal(other)
	return string(asJSON1) == string(asJSON2)
}

// UUID returns the UUID of this contact
func (c *Contact) UUID() ContactUUID { return c.uuid }

// ID returns the numeric ID of this contact
func (c *Contact) ID() ContactID { return c.id }

// SetLanguage sets the language for this contact
func (c *Contact) SetLanguage(lang utils.Language) { c.language = lang }

// Language gets the language for this contact
func (c *Contact) Language() utils.Language { return c.language }

// SetTimezone sets the timezone of this contact
func (c *Contact) SetTimezone(tz *time.Location) {
	c.timezone = tz
}

// Timezone returns the timezone of this contact
func (c *Contact) Timezone() *time.Location { return c.timezone }

// SetCreatedOn sets the created on time of this contact
func (c *Contact) SetCreatedOn(createdOn time.Time) {
	c.createdOn = createdOn
}

// CreatedOn returns the created on time of this contact
func (c *Contact) CreatedOn() time.Time { return c.createdOn }

// SetName sets the name of this contact
func (c *Contact) SetName(name string) { c.name = name }

// Name returns the name of this contact
func (c *Contact) Name() string { return c.name }

// URNs returns the URNs of this contact
func (c *Contact) URNs() URNList { return c.urns }

// AddURN adds a new URN to this contact
func (c *Contact) AddURN(urn *ContactURN) bool {
	if c.HasURN(urn.URN()) {
		return false
	}

	c.urns = append(c.urns, urn)
	return true
}

// HasURN checks whether the contact has the given URN
func (c *Contact) HasURN(urn urns.URN) bool {
	urn = urn.Normalize("")

	for _, u := range c.urns {
		if u.URN().Identity() == urn.Identity() {
			return true
		}
	}
	return false
}

// Fields returns this contact's field values
func (c *Contact) Fields() FieldValues { return c.fields }

// Groups returns the groups that this contact belongs to
func (c *Contact) Groups() *GroupList { return c.groups }

// Reference returns a reference to this contact
func (c *Contact) Reference() *ContactReference {
	if c == nil {
		return nil
	}
	return NewContactReference(c.uuid, c.name)
}

// Format returns a friendly string version of this contact depending on what fields are set
func (c *Contact) Format(env utils.Environment) string {
	// if contact has a name set, use that
	if c.name != "" {
		return c.name
	}

	// otherwise use either id or the higest priority URN depending on the env
	if env.RedactionPolicy() == utils.RedactionPolicyURNs {
		return strconv.Itoa(int(c.id))
	}
	if len(c.urns) > 0 {
		return c.urns[0].URN().Format()
	}

	return ""
}

// Context returns the properties available in expressions
func (c *Contact) Context(env utils.Environment) map[string]types.XValue {
	var urn, timezone types.XValue
	if c.timezone != nil {
		timezone = types.NewXText(c.timezone.String())
	}
	preferredURN := c.PreferredURN()
	if preferredURN != nil {
		urn = preferredURN.ToXValue(env)
	}

	var firstName types.XValue
	names := utils.TokenizeString(c.name)
	if len(names) >= 1 {
		firstName = types.NewXText(names[0])
	}

	return map[string]types.XValue{
		"__default__": types.NewXText(c.Format(env)),
		"uuid":        types.NewXText(string(c.uuid)),
		"id":          types.NewXText(strconv.Itoa(int(c.id))),
		"name":        types.NewXText(c.name),
		"first_name":  firstName,
		"language":    types.NewXText(string(c.language)),
		"timezone":    timezone,
		"created_on":  types.NewXDateTime(c.createdOn),
		"urns":        c.urns.ToXValue(env),
		"urn":         urn,
		"groups":      c.groups.ToXValue(env),
		"fields":      Context(env, c.Fields()),
		"channel":     Context(env, c.PreferredChannel()),
	}
}

// Destination is a sendable channel and URN pair
type Destination struct {
	Channel *Channel
	URN     *ContactURN
}

// ResolveDestinations resolves possible URN/channel destinations
func (c *Contact) ResolveDestinations(all bool) []Destination {
	destinations := []Destination{}

	for _, u := range c.urns {
		channel := c.assets.Channels().GetForURN(u, assets.ChannelRoleSend)
		if channel != nil {
			destinations = append(destinations, Destination{URN: u, Channel: channel})
			if !all {
				break
			}
		}
	}
	return destinations
}

// PreferredURN gets the preferred URN for this contact, i.e. the URN we would use for sending
func (c *Contact) PreferredURN() *ContactURN {
	destinations := c.ResolveDestinations(false)
	if len(destinations) > 0 {
		return destinations[0].URN
	}
	return nil
}

// PreferredChannel gets the preferred channel for this contact, i.e. the channel we would use for sending
func (c *Contact) PreferredChannel() *Channel {
	destinations := c.ResolveDestinations(false)
	if len(destinations) > 0 {
		return destinations[0].Channel
	}
	return nil
}

// UpdatePreferredChannel updates the preferred channel and returns whether any change was made
func (c *Contact) UpdatePreferredChannel(channel *Channel) bool {
	oldURNs := c.urns.clone()

	// setting preferred channel to nil means clearing affinity on all URNs
	if channel == nil {
		for _, urn := range c.urns {
			urn.SetChannel(nil)
		}
	} else {
		priorityURNs := make([]*ContactURN, 0)
		otherURNs := make([]*ContactURN, 0)

		for _, urn := range c.urns {
			// tel URNs can be re-assigned, other URN schemes are considered channel specific
			if urn.URN().Scheme() == urns.TelScheme && channel.SupportsScheme(urns.TelScheme) {
				urn.SetChannel(channel)
			}

			// move any URNs with this channel to the front of the list
			if urn.Channel() == channel {
				priorityURNs = append(priorityURNs, urn)
			} else {
				otherURNs = append(otherURNs, urn)
			}
		}

		c.urns = append(priorityURNs, otherURNs...)
	}

	return !oldURNs.Equal(c.urns)
}

// ReevaluateDynamicGroups reevaluates membership of all dynamic groups for this contact
func (c *Contact) ReevaluateDynamicGroups(env utils.Environment, allGroups *GroupAssets) ([]*Group, []*Group, []error) {
	added := make([]*Group, 0)
	removed := make([]*Group, 0)
	errors := make([]error, 0)

	for _, group := range allGroups.All() {
		if !group.IsDynamic() {
			continue
		}

		qualifies, err := group.CheckDynamicMembership(env, c)
		if err != nil {
			errors = append(errors, err)
		} else if qualifies {
			if c.groups.Add(group) {
				added = append(added, group)
			}
		} else {
			if c.groups.Remove(group) {
				removed = append(removed, group)
			}
		}
	}

	return added, removed, errors
}

// ResolveQueryKey resolves a contact query search key for this contact
func (c *Contact) ResolveQueryKey(env utils.Environment, key string) []interface{} {
	switch key {
	case "name":
		if c.name != "" {
			return []interface{}{c.name}
		}
		return nil
	case "language":
		if c.language != utils.NilLanguage {
			return []interface{}{string(c.language)}
		}
		return nil
	case "created_on":
		return []interface{}{c.createdOn}
	}

	// try as a URN scheme
	if urns.IsValidScheme(key) {
		if env.RedactionPolicy() != utils.RedactionPolicyURNs {
			urnsWithScheme := c.urns.WithScheme(key)
			vals := make([]interface{}, len(urnsWithScheme))
			for i := range urnsWithScheme {
				vals[i] = string(urnsWithScheme[i].URN())
			}
			return vals
		}
		return nil
	}

	// try as a contact field
	nativeValue := c.fields[key].QueryValue()
	if nativeValue == nil {
		return nil
	}
	return []interface{}{nativeValue}
}

var _ contactql.Queryable = (*Contact)(nil)

// ContactReference is used to reference a contact
type ContactReference struct {
	UUID ContactUUID `json:"uuid" validate:"required,uuid4"`
	Name string      `json:"name"`
}

// NewContactReference creates a new contact reference with the given UUID and name
func NewContactReference(uuid ContactUUID, name string) *ContactReference {
	return &ContactReference{UUID: uuid, Name: name}
}

// Type returns the name of the asset type
func (r *ContactReference) Type() string {
	return "contact"
}

// Identity returns the unique identity of the asset
func (r *ContactReference) Identity() string {
	return string(r.UUID)
}

// Variable returns whether this a variable (vs concrete) reference
func (r *ContactReference) Variable() bool {
	return r.Identity() == ""
}

func (r *ContactReference) String() string {
	return fmt.Sprintf("%s[uuid=%s,name=%s]", r.Type(), r.Identity(), r.Name)
}

var _ assets.Reference = (*ContactReference)(nil)

//------------------------------------------------------------------------------------------
// JSON Encoding / Decoding
//------------------------------------------------------------------------------------------

type contactEnvelope struct {
	UUID      ContactUUID              `json:"uuid" validate:"required,uuid4"`
	ID        ContactID                `json:"id,omitempty"`
	Name      string                   `json:"name,omitempty"`
	Language  utils.Language           `json:"language,omitempty"`
	Timezone  string                   `json:"timezone,omitempty"`
	CreatedOn time.Time                `json:"created_on" validate:"required"`
	URNs      []urns.URN               `json:"urns,omitempty" validate:"dive,urn"`
	Groups    []*assets.GroupReference `json:"groups,omitempty" validate:"dive"`
	Fields    map[string]*Value        `json:"fields,omitempty"`
}

// ReadContact decodes a contact from the passed in JSON
func ReadContact(sa SessionAssets, data json.RawMessage, missing assets.MissingCallback) (*Contact, error) {
	var envelope contactEnvelope
	var err error

	if err := utils.UnmarshalAndValidate(data, &envelope); err != nil {
		return nil, errors.Wrap(err, "unable to read contact")
	}

	c := &Contact{
		uuid:      envelope.UUID,
		id:        envelope.ID,
		name:      envelope.Name,
		language:  envelope.Language,
		createdOn: envelope.CreatedOn,
		assets:    sa,
	}

	if envelope.Timezone != "" {
		if c.timezone, err = time.LoadLocation(envelope.Timezone); err != nil {
			return nil, err
		}
	}

	if envelope.URNs == nil {
		c.urns = make(URNList, 0)
	} else {
		if c.urns, err = ReadURNList(sa, envelope.URNs, missing); err != nil {
			return nil, errors.Wrap(err, "error reading urns")
		}
	}

	if envelope.Groups == nil {
		c.groups = NewGroupList([]*Group{})
	} else {
		groups := make([]*Group, 0, len(envelope.Groups))
		for _, g := range envelope.Groups {
			group := sa.Groups().Get(g.UUID)
			if group == nil {
				missing(g, nil)
			} else {
				groups = append(groups, group)
			}
		}
		c.groups = NewGroupList(groups)
	}

	if c.fields, err = NewFieldValues(sa, envelope.Fields, missing); err != nil {
		return nil, errors.Wrap(err, "error reading fields")
	}

	return c, nil
}

// MarshalJSON marshals this contact into JSON
func (c *Contact) MarshalJSON() ([]byte, error) {
	ce := &contactEnvelope{
		Name:      c.name,
		UUID:      c.uuid,
		ID:        c.id,
		Language:  c.language,
		CreatedOn: c.createdOn,
	}

	ce.URNs = c.urns.RawURNs()
	if c.timezone != nil {
		ce.Timezone = c.timezone.String()
	}

	ce.Groups = make([]*assets.GroupReference, c.groups.Count())
	for i, group := range c.groups.All() {
		ce.Groups[i] = group.Reference()
	}

	ce.Fields = make(map[string]*Value)
	for _, v := range c.fields {
		if v != nil {
			ce.Fields[v.field.Key()] = v.Value
		}
	}

	return json.Marshal(ce)
}
