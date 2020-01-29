package flows_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/urns"
	"github.com/nyaruka/goflow/assets"
	"github.com/nyaruka/goflow/assets/static"
	"github.com/nyaruka/goflow/contactql"
	"github.com/nyaruka/goflow/envs"
	"github.com/nyaruka/goflow/excellent/types"
	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/flows/engine"
	"github.com/nyaruka/goflow/test"
	"github.com/nyaruka/goflow/utils/uuids"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContact(t *testing.T) {
	source, err := static.NewSource([]byte(`{
		"channels": [
			{
				"uuid": "294a14d4-c998-41e5-a314-5941b97b89d7",
				"name": "My Android Phone",
				"address": "+12345671111",
				"schemes": ["tel"],
				"roles": ["send", "receive"]
			}
		]
	}`))
	require.NoError(t, err)

	sa, err := engine.NewSessionAssets(source, nil)
	require.NoError(t, err)

	android := sa.Channels().Get("294a14d4-c998-41e5-a314-5941b97b89d7")

	env := envs.NewBuilder().Build()

	uuids.SetGenerator(uuids.NewSeededGenerator(1234))
	defer uuids.SetGenerator(uuids.DefaultGenerator)

	contact, _ := flows.NewContact(
		sa, flows.ContactUUID(uuids.New()), flows.ContactID(12345), "Joe Bloggs", envs.Language("eng"),
		nil, time.Now(), nil, nil, nil,
	)

	assert.Equal(t, flows.URNList{}, contact.URNs())
	assert.Nil(t, contact.PreferredChannel())

	contact.SetTimezone(env.Timezone())
	contact.SetCreatedOn(time.Date(2017, 12, 15, 10, 0, 0, 0, time.UTC))
	contact.AddURN(flows.NewContactURN(urns.URN("tel:+16364646466?channel=294a14d4-c998-41e5-a314-5941b97b89d7"), nil))
	contact.AddURN(flows.NewContactURN(urns.URN("twitter:joey"), nil))

	assert.Equal(t, "Joe Bloggs", contact.Name())
	assert.Equal(t, flows.ContactID(12345), contact.ID())
	assert.Equal(t, env.Timezone(), contact.Timezone())
	assert.Equal(t, envs.Language("eng"), contact.Language())
	assert.Equal(t, android, contact.PreferredChannel())
	assert.True(t, contact.HasURN("tel:+16364646466"))
	assert.False(t, contact.HasURN("tel:+16300000000"))

	test.AssertXEqual(t, types.NewXObject(map[string]types.XValue{
		"ext":       nil,
		"facebook":  nil,
		"fcm":       nil,
		"freshchat": nil,
		"jiochat":   nil,
		"line":      nil,
		"mailto":    nil,
		"tel":       flows.NewContactURN(urns.URN("tel:+16364646466?channel=294a14d4-c998-41e5-a314-5941b97b89d7"), nil).ToXValue(env),
		"telegram":  nil,
		"twitter":   flows.NewContactURN(urns.URN("twitter:joey"), nil).ToXValue(env),
		"twitterid": nil,
		"viber":     nil,
		"wechat":    nil,
		"whatsapp":  nil,
	}), flows.ContextFunc(env, contact.URNs().MapContext))

	clone := contact.Clone()
	assert.Equal(t, "Joe Bloggs", clone.Name())
	assert.Equal(t, flows.ContactID(12345), clone.ID())
	assert.Equal(t, env.Timezone(), clone.Timezone())
	assert.Equal(t, envs.Language("eng"), clone.Language())
	assert.Equal(t, android, contact.PreferredChannel())

	// can also clone a null contact!
	mrNil := (*flows.Contact)(nil)
	assert.Nil(t, mrNil.Clone())

	test.AssertXEqual(t, types.NewXObject(map[string]types.XValue{
		"__default__": types.NewXText("Joe Bloggs"),
		"channel":     flows.Context(env, android),
		"created_on":  types.NewXDateTime(contact.CreatedOn()),
		"fields":      flows.Context(env, contact.Fields()),
		"first_name":  types.NewXText("Joe"),
		"groups":      contact.Groups().ToXValue(env),
		"id":          types.NewXText("12345"),
		"language":    types.NewXText("eng"),
		"name":        types.NewXText("Joe Bloggs"),
		"timezone":    types.NewXText("UTC"),
		"urn":         contact.URNs()[0].ToXValue(env),
		"urns":        contact.URNs().ToXValue(env),
		"uuid":        types.NewXText(string(contact.UUID())),
	}), flows.Context(env, contact))
}

func TestContactFormat(t *testing.T) {
	env := envs.NewBuilder().Build()
	sa, _ := engine.NewSessionAssets(static.NewEmptySource(), nil)

	// name takes precedence if set
	contact := flows.NewEmptyContact(sa, "Joe", envs.NilLanguage, nil)
	contact.AddURN(flows.NewContactURN(urns.URN("twitter:joey"), nil))
	assert.Equal(t, "Joe", contact.Format(env))

	// if not we fallback to URN
	contact, _ = flows.NewContact(
		sa, flows.ContactUUID(uuids.New()), flows.ContactID(1234), "", envs.NilLanguage, nil, time.Now(),
		nil, nil, nil,
	)
	contact.AddURN(flows.NewContactURN(urns.URN("twitter:joey"), nil))
	assert.Equal(t, "joey", contact.Format(env))

	anonEnv := envs.NewBuilder().WithRedactionPolicy(envs.RedactionPolicyURNs).Build()

	// unless URNs are redacted
	assert.Equal(t, "1234", contact.Format(anonEnv))

	// if we don't have name or URNs, then empty string
	contact = flows.NewEmptyContact(sa, "", envs.NilLanguage, nil)
	assert.Equal(t, "", contact.Format(env))
}

func TestContactSetPreferredChannel(t *testing.T) {
	sa, _ := engine.NewSessionAssets(static.NewEmptySource(), nil)
	roles := []assets.ChannelRole{assets.ChannelRoleSend}

	android := test.NewTelChannel("Android", "+250961111111", roles, nil, "RW", nil)
	twitter1 := test.NewChannel("Twitter", "nyaruka", []string{"twitter", "twitterid"}, roles, nil)
	twitter2 := test.NewChannel("Twitter", "nyaruka", []string{"twitter", "twitterid"}, roles, nil)

	contact := flows.NewEmptyContact(sa, "Joe", envs.NilLanguage, nil)
	contact.AddURN(flows.NewContactURN(urns.URN("twitter:joey"), nil))
	contact.AddURN(flows.NewContactURN(urns.URN("tel:+12345678999"), nil))
	contact.AddURN(flows.NewContactURN(urns.URN("tel:+18005555777"), nil))

	contact.UpdatePreferredChannel(android)

	// tel channels should be re-assigned to that channel, and moved to front of list
	assert.Equal(t, urns.URN("tel:+12345678999?channel="+string(android.UUID())), contact.URNs()[0].URN())
	assert.Equal(t, android, contact.URNs()[0].Channel())
	assert.Equal(t, urns.URN("tel:+18005555777?channel="+string(android.UUID())), contact.URNs()[1].URN())
	assert.Equal(t, android, contact.URNs()[1].Channel())
	assert.Equal(t, urns.URN("twitter:joey"), contact.URNs()[2].URN())
	assert.Nil(t, contact.URNs()[2].Channel())

	// same only applies to URNs of other schemes if they don't have a channel already
	contact.UpdatePreferredChannel(twitter1)
	assert.Equal(t, urns.URN("twitter:joey?channel="+string(twitter1.UUID())), contact.URNs()[0].URN())

	contact.UpdatePreferredChannel(twitter2)
	assert.Equal(t, urns.URN("twitter:joey?channel="+string(twitter1.UUID())), contact.URNs()[0].URN())

	// if they are already associated with the channel, then they become the preferred URN
	contact.UpdatePreferredChannel(android)
	contact.UpdatePreferredChannel(twitter1)

	assert.Equal(t, urns.URN("twitter:joey?channel="+string(twitter1.UUID())), contact.URNs()[0].URN())
	assert.Equal(t, twitter1, contact.URNs()[0].Channel())
}

func TestReevaluateDynamicGroups(t *testing.T) {
	session, _, err := test.CreateTestSession("http://localhost", envs.RedactionPolicyNone)
	require.NoError(t, err)

	env := session.Runs()[0].Environment()

	gender := test.NewField("gender", "Gender", assets.FieldTypeText)
	age := test.NewField("age", "Age", assets.FieldTypeNumber)

	fieldSet := flows.NewFieldAssets([]assets.Field{gender.Asset(), age.Asset()})

	males := test.NewGroup("Males", `gender="M"`)
	old := test.NewGroup("Old", `age>30`)
	english := test.NewGroup("English", `language=eng`)
	spanish := test.NewGroup("Español", `language=spa`)
	lastYear := test.NewGroup("Old", `created_on <= 2017-12-31`)
	tel1800 := test.NewGroup("Tel with 1800", `tel ~ 1800`)
	twitterCrazies := test.NewGroup("Twitter Crazies", `twitter ~ crazy`)
	broken := test.NewGroup("Broken", `xyz="X"`) // no such field
	groups := []*flows.Group{males, old, english, spanish, lastYear, tel1800, twitterCrazies, broken}

	contact := flows.NewEmptyContact(session.Assets(), "Joe", "eng", nil)
	contact.AddURN(flows.NewContactURN(urns.URN("tel:+12345678999"), nil))

	memberships, errors := evaluateGroups(t, env, contact, groups, fieldSet)
	assert.Equal(t, []*flows.Group{english}, memberships)
	assert.Equal(t, []*flows.Group{broken}, errors)

	contact.SetLanguage(envs.Language("spa"))
	contact.AddURN(flows.NewContactURN(urns.URN("twitter:crazy_joe"), nil))
	contact.AddURN(flows.NewContactURN(urns.URN("tel:+18005555777"), nil))

	genderValue := contact.Fields().Parse(env, fieldSet, gender, "M")
	contact.Fields().Set(gender, genderValue)

	ageValue := contact.Fields().Parse(env, fieldSet, age, "37")
	contact.Fields().Set(age, ageValue)

	contact.SetCreatedOn(time.Date(2017, 12, 15, 10, 0, 0, 0, time.UTC))

	memberships, errors = evaluateGroups(t, env, contact, groups, fieldSet)
	assert.Equal(t, []*flows.Group{males, old, spanish, lastYear, tel1800, twitterCrazies}, memberships)
	assert.Equal(t, []*flows.Group{broken}, errors)
}

func TestReevaluateDynamicGroupsWithURNRedaction(t *testing.T) {
	session, _, err := test.CreateTestSession("http://localhost", envs.RedactionPolicyURNs)
	require.NoError(t, err)

	env := session.Runs()[0].Environment()

	gender := test.NewField("gender", "Gender", assets.FieldTypeText)
	age := test.NewField("age", "Age", assets.FieldTypeNumber)

	fieldSet := flows.NewFieldAssets([]assets.Field{gender.Asset(), age.Asset()})

	males := test.NewGroup("Males", `gender="M"`)
	tel1800 := test.NewGroup("Tel with 1800", `tel ~ 1800`)
	twitterCrazies := test.NewGroup("Twitter Crazies", `twitter ~ crazy`)
	groups := []*flows.Group{males, tel1800, twitterCrazies}

	contact := flows.NewEmptyContact(session.Assets(), "Joe", "eng", nil)
	contact.AddURN(flows.NewContactURN(urns.URN("tel:+12345678999"), nil))

	memberships, errors := evaluateGroups(t, env, contact, groups, fieldSet)
	assert.Equal(t, []*flows.Group{}, memberships)
	assert.Equal(t, []*flows.Group{tel1800, twitterCrazies}, errors) // both groups with URN references error

	contact.AddURN(flows.NewContactURN(urns.URN("twitter:crazy_joe"), nil))
	contact.AddURN(flows.NewContactURN(urns.URN("tel:+18005555777"), nil))

	genderValue := contact.Fields().Parse(env, fieldSet, gender, "M")
	contact.Fields().Set(gender, genderValue)

	memberships, errors = evaluateGroups(t, env, contact, groups, fieldSet)
	assert.Equal(t, []*flows.Group{males}, memberships)
	assert.Equal(t, []*flows.Group{tel1800, twitterCrazies}, errors)
}

func evaluateGroups(t *testing.T, env envs.Environment, contact *flows.Contact, groups []*flows.Group, fields *flows.FieldAssets) ([]*flows.Group, []*flows.Group) {
	memberships := make([]*flows.Group, 0)
	errors := make([]*flows.Group, 0)

	for _, group := range groups {
		isMember, err := group.CheckDynamicMembership(env, contact, fields)
		if err != nil {
			errors = append(errors, group)
		} else if isMember {
			memberships = append(memberships, group)
		}
	}
	return memberships, errors
}

func TestContactEqual(t *testing.T) {
	session, _, err := test.CreateTestSession("http://localhost", envs.RedactionPolicyNone)
	require.NoError(t, err)

	contact1JSON := []byte(`{
		"uuid": "ba96bf7f-bc2a-4873-a7c7-254d1927c4e3",
		"id": 1234567,
		"created_on": "2000-01-01T00:00:00.000000000-00:00",
		"fields": {
			"gender": {"text": "Male"}
		},
		"language": "eng",
		"name": "Ben Haggerty",
		"timezone": "America/Guayaquil",
		"urns": ["tel:+12065551212"]
	}`)

	contact1, err := flows.ReadContact(session.Assets(), contact1JSON, assets.PanicOnMissing)
	require.NoError(t, err)

	contact2, err := flows.ReadContact(session.Assets(), contact1JSON, assets.PanicOnMissing)
	require.NoError(t, err)

	assert.True(t, contact1.Equal(contact2))
	assert.True(t, contact2.Equal(contact1))
	assert.True(t, contact1.Equal(contact1.Clone()))

	// marshal and unmarshal contact 1 again
	contact1JSON, err = json.Marshal(contact1)
	require.NoError(t, err)
	contact1, err = flows.ReadContact(session.Assets(), contact1JSON, assets.PanicOnMissing)
	require.NoError(t, err)

	assert.True(t, contact1.Equal(contact2))

	contact2.SetLanguage(envs.NilLanguage)
	assert.False(t, contact1.Equal(contact2))
}

func TestContactQuery(t *testing.T) {
	session, _, err := test.CreateTestSession("", envs.RedactionPolicyNone)
	require.NoError(t, err)

	contactJSON := []byte(`{
		"uuid": "ba96bf7f-bc2a-4873-a7c7-254d1927c4e3",
		"id": 1234567,
		"name": "Ben Haggerty",
		"fields": {
			"gender": {"text": "Male"}
		},
		"language": "eng",
		"timezone": "America/Guayaquil",
		"urns": ["tel:+12065551212", "twitter:ewok"],
		"created_on": "2020-01-24T13:24:30.000000000-00:00"
	}`)

	contact, err := flows.ReadContact(session.Assets(), contactJSON, assets.PanicOnMissing)
	require.NoError(t, err)

	testCases := []struct {
		query  string
		result bool
	}{
		{`name = "Ben Haggerty"`, true},
		{`name = "Joe X"`, false},
		{`name != "Joe X"`, true},
		{`name != ""`, true},
		{`name = ""`, false},
		{`name ~ Ben`, true},
		{`name ~ Joe`, false},

		{`id = 1234567`, true},
		{`id = 5678889`, false},

		{`language = ENG`, true},
		{`language = FRA`, false},
		{`language = ""`, false},
		{`language != ""`, true},

		{`created_on = 24-01-2020`, true},
		{`created_on = 25-01-2020`, false},
		{`created_on > 22-01-2020`, true},
		{`created_on > 26-01-2020`, false},

		{`tel = +12065551212`, true},
		{`tel = +13065551212`, false},
		{`tel ~ 555`, true},
		{`tel ~ 666`, false},

		{`twitter = ewok`, true},
		{`twitter = nicp`, false},
		{`twitter ~ wok`, true},
		{`twitter ~ EWO`, true},
		{`twitter ~ ijk`, false},

		{`urn = +12065551212`, true},
		{`urn = ewok`, true},
		{`urn = +13065551212`, false},
		{`urn ~ 555`, true},
		{`urn ~ 666`, false},
	}

	for _, tc := range testCases {
		query, err := contactql.ParseQuery(tc.query, envs.RedactionPolicyNone, session.Assets().Fields().Resolve)
		require.NoError(t, err, "unexpected error parsing '%s'", tc.query)

		result, err := contactql.EvaluateQuery(session.Environment(), query, contact)
		require.NoError(t, err, "unexpected error evaluating '%s'", tc.query)

		assert.Equal(t, tc.result, result, "unexpected result for '%s' ('%s')", tc.query, query.String())
	}
}
