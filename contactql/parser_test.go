package contactql

import (
	"testing"
	"time"

	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/assets/static/types"
	"github.com/greatnonprofits-nfp/goflow/envs"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		text   string
		parsed string
		err    string
		redact envs.RedactionPolicy
	}{
		// implicit conditions
		{`will`, "name~will", "", envs.RedactionPolicyNone},
		{`0123456566`, "tel~0123456566", "", envs.RedactionPolicyNone},
		{`+0123456566`, "tel~0123456566", "", envs.RedactionPolicyNone},
		{`0123-456-566`, "tel~0123456566", "", envs.RedactionPolicyNone},

		// implicit conditions with URN redaction
		{`will`, "name~will", "", envs.RedactionPolicyURNs},
		{`0123456566`, "id=123456566", "", envs.RedactionPolicyURNs},
		{`+0123456566`, "id=123456566", "", envs.RedactionPolicyURNs},
		{`0123-456-566`, "name~0123-456-566", "", envs.RedactionPolicyURNs},

		{`will felix`, "AND(name~will, name~felix)", "", envs.RedactionPolicyNone},     // implicit AND
		{`will and felix`, "AND(name~will, name~felix)", "", envs.RedactionPolicyNone}, // explicit AND
		{`will or felix or matt`, "OR(OR(name~will, name~felix), name~matt)", "", envs.RedactionPolicyNone},
		{`Name=will`, "name=will", "", envs.RedactionPolicyNone},
		{`Name ~ "felix"`, "name~felix", "", envs.RedactionPolicyNone},
		{`name is ""`, `name=""`, "", envs.RedactionPolicyNone},          // is not set
		{`name != ""`, `name!=""`, "", envs.RedactionPolicyNone},         // is set
		{`name != "felix"`, `name!=felix`, "", envs.RedactionPolicyNone}, // is set
		{`name=will or Name ~ "felix"`, "OR(name=will, name~felix)", "", envs.RedactionPolicyNone},
		{`Name is will or Name has felix`, "OR(name=will, name~felix)", "", envs.RedactionPolicyNone}, // comparator aliases
		{`will or Name ~ "felix"`, "OR(name~will, name~felix)", "", envs.RedactionPolicyNone},

		{`mailto = user@example.com`, "mailto=user@example.com", "", envs.RedactionPolicyNone},
		{`MAILTO ~ user@example.com`, "mailto~user@example.com", "", envs.RedactionPolicyNone},

		{`mailto = user@example.com`, "", "cannot query on redacted URNs", envs.RedactionPolicyURNs},
		{`MAILTO ~ user@example.com`, "", "cannot query on redacted URNs", envs.RedactionPolicyURNs},

		// boolean operator precedence is AND before OR, even when AND is implicit
		{`will and felix or matt amber`, "OR(AND(name~will, name~felix), AND(name~matt, name~amber))", "", envs.RedactionPolicyNone},

		// boolean combinations can themselves be combined
		{
			`(Age < 18 and Gender = "male") or (Age > 18 and Gender = "female")`,
			"OR(AND(age<18, gender=male), AND(age>18, gender=female))",
			"",
			envs.RedactionPolicyNone,
		},

		{`xyz != ""`, "", "can't resolve 'xyz' to attribute, scheme or field", envs.RedactionPolicyNone},

		{`name = "O\"Leary"`, `name=O"Leary`, "", envs.RedactionPolicyNone}, // string unquoting
	}

	fields := map[string]assets.Field{
		"age":    types.NewField(assets.FieldUUID("f1b5aea6-6586-41c7-9020-1a6326cc6565"), "age", "Age", assets.FieldTypeNumber),
		"gender": types.NewField(assets.FieldUUID("d66a7823-eada-40e5-9a3a-57239d4690bf"), "gender", "Gender", assets.FieldTypeText),
	}
	fieldResolver := func(key string) assets.Field { return fields[key] }

	for _, tc := range tests {
		parsed, err := ParseQuery(tc.text, tc.redact, fieldResolver)
		if tc.err != "" {
			assert.EqualError(t, err, tc.err, "error mismatch for '%s'", tc.text)
			assert.Nil(t, parsed)
		} else {
			assert.NoError(t, err, "unexpected error for '%s'", tc.text)
			assert.Equal(t, tc.parsed, parsed.String(), "parse mismatch for '%s'", tc.text)
		}
	}
}

type TestQueryable struct{}

func (t *TestQueryable) QueryProperty(env envs.Environment, key string, propType PropertyType) []interface{} {
	switch key {
	case "tel":
		return []interface{}{"+59313145145"}
	case "twitter":
		return []interface{}{"bob_smith"}
	case "whatsapp":
		return []interface{}{}
	case "gender":
		return []interface{}{"male"}
	case "age":
		return []interface{}{decimal.NewFromFloat(36)}
	case "dob":
		return []interface{}{time.Date(1981, 5, 28, 13, 30, 23, 0, time.UTC)}
	case "state":
		return []interface{}{"Kigali"}
	case "district":
		return []interface{}{"Gasabo"}
	case "ward":
		return []interface{}{"Ndera"}
	case "empty":
		return []interface{}{""}
	case "nope":
		return []interface{}{t}
	}
	return nil
}

func TestEvaluateQuery(t *testing.T) {
	env := envs.NewBuilder().Build()
	testObj := &TestQueryable{}

	tests := []struct {
		query  string
		result bool
	}{
		// URN condition
		{`tel = +59313145145`, true},
		{`tel has 45145`, true},
		{`tel ~ 33333`, false},
		{`TWITTER IS bob_smith`, true},
		{`twitter = jim_smith`, false},
		{`twitter ~ smith`, true},
		{`whatsapp = 4533343`, false},

		// text field condition
		{`Gender = male`, true},
		{`Gender is MALE`, true},
		{`gender = "female"`, false},
		{`gender != "female"`, true},
		{`gender != "male"`, false},
		{`empty != "male"`, true}, // this is true because "" is not "male"
		{`gender != ""`, true},

		// number field condition
		{`age = 36`, true},
		{`age is 35`, false},
		{`age > 36`, false},
		{`age > 35`, true},
		{`age >= 36`, true},
		{`age < 36`, false},
		{`age < 37`, true},
		{`age <= 36`, true},

		// datetime field condition
		{`dob = 1981/05/28`, true},
		{`dob > 1981/05/28`, false},
		{`dob > 1981/05/27`, true},
		{`dob >= 1981/05/28`, true},
		{`dob >= 1981/05/29`, false},
		{`dob < 1981/05/28`, false},
		{`dob < 1981/05/29`, true},
		{`dob <= 1981/05/28`, true},
		{`dob <= 1981/05/27`, false},

		// location field condition
		{`state = kigali`, true},
		{`state = "kigali"`, true},
		{`state = "NY"`, false},
		{`state ~ KIG`, true},
		{`state ~ NY`, false},
		{`district = "GASABO"`, true},
		{`district = "Brooklyn"`, false},
		{`district ~ SAB`, true},
		{`district ~ BRO`, false},
		{`ward = ndera`, true},
		{`ward = solano`, false},
		{`ward ~ era`, true},
		{`ward != ndera`, false},
		{`ward != solano`, true},

		// existence
		{`age = ""`, false},
		{`age != ""`, true},
		{`xyz = ""`, true},
		{`xyz != ""`, false},
		{`age != "" AND xyz != ""`, false},
		{`age != "" OR xyz != ""`, true},

		// boolean combinations
		{`age = 36 AND gender = male`, true},
		{`(age = 36) AND (gender = male)`, true},
		{`age = 36 AND gender = female`, false},
		{`age = 36 OR gender = female`, true},
		{`age = 35 OR gender = female`, false},
		{`(age = 36 OR gender = female) AND age > 35`, true},
	}

	fields := map[string]assets.Field{
		"age":      types.NewField(assets.FieldUUID("f1b5aea6-6586-41c7-9020-1a6326cc6565"), "age", "Age", assets.FieldTypeNumber),
		"dob":      types.NewField(assets.FieldUUID("3810a485-3fda-4011-a589-7320c0b8dbef"), "dob", "DOB", assets.FieldTypeDatetime),
		"gender":   types.NewField(assets.FieldUUID("d66a7823-eada-40e5-9a3a-57239d4690bf"), "gender", "Gender", assets.FieldTypeText),
		"state":    types.NewField(assets.FieldUUID("369be3e2-0186-4e5d-93c4-6264736588f8"), "state", "State", assets.FieldTypeState),
		"district": types.NewField(assets.FieldUUID("e52f34ad-a5a7-4855-9040-05a910a75f57"), "district", "District", assets.FieldTypeDistrict),
		"ward":     types.NewField(assets.FieldUUID("e9e738ce-617d-4c61-bfce-3d3b55cfe3dd"), "ward", "Ward", assets.FieldTypeWard),
		"empty":    types.NewField(assets.FieldUUID("023f733d-ce00-4a61-96e4-b411987028ea"), "empty", "Empty", assets.FieldTypeText),
		"xyz":      types.NewField(assets.FieldUUID("81e25783-a1d8-42b9-85e4-68c7ab2df39d"), "xyz", "XYZ", assets.FieldTypeText),
	}
	fieldResolver := func(key string) assets.Field { return fields[key] }

	for _, test := range tests {
		parsed, err := ParseQuery(test.query, envs.RedactionPolicyNone, fieldResolver)
		assert.NoError(t, err, "unexpected error parsing '%s'", test.query)

		actualResult, err := EvaluateQuery(env, parsed, testObj)
		assert.NoError(t, err, "unexpected error evaluating '%s'", test.query)
		assert.Equal(t, test.result, actualResult, "unexpected result for '%s'", test.query)
	}
}

func TestParsingErrors(t *testing.T) {
	_, err := ParseQuery("name = ", envs.RedactionPolicyNone, nil)
	assert.EqualError(t, err, "mismatched input '<EOF>' expecting {TEXT, STRING}")
}

func TestEvaluationErrors(t *testing.T) {
	env := envs.NewBuilder().Build()
	testObj := &TestQueryable{}

	tests := []struct {
		query  string
		errMsg string
	}{
		{`gender > Male`, "can't query text fields with >"},
		{`age = 3X`, "can't convert '3X' to a number"},
		{`age ~ 32`, "can't query number fields with ~"},
		{`dob = 32`, "string '32' couldn't be parsed as a date"},
		{`dob = 32 AND name = Bob`, "string '32' couldn't be parsed as a date"},
		{`name = Bob OR dob = 32`, "string '32' couldn't be parsed as a date"},
		{`dob ~ 2018-12-31`, "can't query datetime fields with ~"},
	}

	fields := map[string]assets.Field{
		"age":    types.NewField(assets.FieldUUID("f1b5aea6-6586-41c7-9020-1a6326cc6565"), "age", "Age", assets.FieldTypeNumber),
		"dob":    types.NewField(assets.FieldUUID("3810a485-3fda-4011-a589-7320c0b8dbef"), "dob", "DOB", assets.FieldTypeDatetime),
		"gender": types.NewField(assets.FieldUUID("d66a7823-eada-40e5-9a3a-57239d4690bf"), "gender", "Gender", assets.FieldTypeText),
	}
	fieldResolver := func(key string) assets.Field { return fields[key] }

	for _, test := range tests {
		parsed, err := ParseQuery(test.query, envs.RedactionPolicyNone, fieldResolver)
		assert.NoError(t, err, "unexpected error parsing '%s'", test.query)

		actualResult, err := EvaluateQuery(env, parsed, testObj)
		assert.EqualError(t, err, test.errMsg, "unexpected error evaluating '%s'", test.query)
		assert.False(t, actualResult, "unexpected non-false result for '%s'", test.query)
	}
}
