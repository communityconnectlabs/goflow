package contactql_test

import (
	"testing"

	"github.com/nyaruka/goflow/assets"
	"github.com/nyaruka/goflow/assets/static/types"
	"github.com/nyaruka/goflow/contactql"
	"github.com/nyaruka/goflow/envs"

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
		{`Will`, `name ~ "Will"`, "", envs.RedactionPolicyNone},
		{`wil`, `name ~ "wil"`, "", envs.RedactionPolicyNone},
		{`wi`, `name ~ "wi"`, "", envs.RedactionPolicyNone},
		{`w`, `name = "w"`, "", envs.RedactionPolicyNone}, // don't have at least 1 token of >= 2 chars
		{`w me`, `name = "w" AND name ~ "me"`, "", envs.RedactionPolicyNone},
		{`w m`, `name = "w" AND name = "m"`, "", envs.RedactionPolicyNone},
		{`tel:+0123456566`, `tel = +0123456566`, "", envs.RedactionPolicyNone},
		{`twitter:bobby`, `twitter = "bobby"`, "", envs.RedactionPolicyNone},
		{`0123456566`, `tel ~ 0123456566`, "", envs.RedactionPolicyNone}, // righthand side looks like a phone number
		{`+0123456566`, `tel ~ 0123456566`, "", envs.RedactionPolicyNone},
		{`0123-456-566`, `tel ~ 0123456566`, "", envs.RedactionPolicyNone},
		{`566`, `name ~ 566`, "", envs.RedactionPolicyNone}, // too short to be a phone number

		// implicit conditions with URN redaction
		{`will`, `name ~ "will"`, "", envs.RedactionPolicyURNs},
		{`tel:+0123456566`, `name ~ "tel:+0123456566"`, "", envs.RedactionPolicyURNs},
		{`twitter:bobby`, `name ~ "twitter:bobby"`, "", envs.RedactionPolicyURNs},
		{`0123456566`, `id = 123456566`, "", envs.RedactionPolicyURNs},
		{`+0123456566`, `id = 123456566`, "", envs.RedactionPolicyURNs},
		{`0123-456-566`, `name ~ "0123-456-566"`, "", envs.RedactionPolicyURNs},

		// explicit conditions on name
		{`Name=will`, `name = "will"`, "", envs.RedactionPolicyNone},
		{`Name ~ "felix"`, `name ~ "felix"`, "", envs.RedactionPolicyNone},
		{`Name HAS "Felix"`, `name ~ "Felix"`, "", envs.RedactionPolicyNone},
		{`name is ""`, `name = ""`, "", envs.RedactionPolicyNone},            // is not set
		{`name != ""`, `name != ""`, "", envs.RedactionPolicyNone},           // is set
		{`name != "felix"`, `name != "felix"`, "", envs.RedactionPolicyNone}, // is not equal to value
		{`Name ~ ""`, ``, "value must contain a word of at least 2 characters long for a contains condition on name", envs.RedactionPolicyNone},

		// explicit conditions on URN
		{`tel=""`, `tel = ""`, "", envs.RedactionPolicyNone},
		{`tel!=""`, `tel != ""`, "", envs.RedactionPolicyNone},
		{`tel IS 233`, `tel = 233`, "", envs.RedactionPolicyNone},
		{`tel HAS 233`, `tel ~ 233`, "", envs.RedactionPolicyNone},
		{`tel ~ 23`, ``, "value must be least 3 characters long for a contains condition on a URN", envs.RedactionPolicyNone},
		{`mailto = user@example.com`, `mailto = "user@example.com"`, "", envs.RedactionPolicyNone},
		{`MAILTO ~ user@example.com`, `mailto ~ "user@example.com"`, "", envs.RedactionPolicyNone},
		{`URN=ewok`, `urn = "ewok"`, "", envs.RedactionPolicyNone},

		// explicit conditions on URN with URN redaction
		{`tel = 233`, ``, "cannot query on redacted URNs", envs.RedactionPolicyURNs},
		{`tel ~ 233`, ``, "cannot query on redacted URNs", envs.RedactionPolicyURNs},
		{`mailto = user@example.com`, ``, "cannot query on redacted URNs", envs.RedactionPolicyURNs},
		{`MAILTO ~ user@example.com`, ``, "cannot query on redacted URNs", envs.RedactionPolicyURNs},
		{`URN=ewok`, ``, "cannot query on redacted URNs", envs.RedactionPolicyURNs},

		// field conditions
		{`Age IS 18`, `age = 18`, "", envs.RedactionPolicyNone},
		{`AGE != ""`, `age != ""`, "", envs.RedactionPolicyNone},
		{`age ~ 34`, ``, "contains conditions can only be used with name or URN values", envs.RedactionPolicyNone},
		{`gender ~ M`, ``, "contains conditions can only be used with name or URN values", envs.RedactionPolicyNone},

		// lt/lte/gt/gte comparisons
		{`Age > "18"`, `age > 18`, "", envs.RedactionPolicyNone},
		{`Age >= 18`, `age >= 18`, "", envs.RedactionPolicyNone},
		{`Age < 18`, `age < 18`, "", envs.RedactionPolicyNone},
		{`Age <= 18`, `age <= 18`, "", envs.RedactionPolicyNone},
		{`DOB > "27-01-2020"`, `dob > "27-01-2020"`, "", envs.RedactionPolicyNone},
		{`DOB >= 27-01-2020`, `dob >= "27-01-2020"`, "", envs.RedactionPolicyNone},
		{`DOB < 27/01/2020`, `dob < "27/01/2020"`, "", envs.RedactionPolicyNone},
		{`DOB <= 27.01.2020`, `dob <= "27.01.2020"`, "", envs.RedactionPolicyNone},
		{`name > Will`, ``, "comparisons with > can only be used with date and number fields", envs.RedactionPolicyNone},
		{`tel < 23425`, ``, "comparisons with < can only be used with date and number fields", envs.RedactionPolicyNone},

		// implicit combinations
		{`will felix`, `name ~ "will" AND name ~ "felix"`, "", envs.RedactionPolicyNone},

		// explicit combinations...
		{`will and felix`, `name ~ "will" AND name ~ "felix"`, "", envs.RedactionPolicyNone}, // explicit AND
		{`will or felix or matt`, `(name ~ "will" OR name ~ "felix") OR name ~ "matt"`, "", envs.RedactionPolicyNone},
		{`name=will or Name ~ "felix"`, `name = "will" OR name ~ "felix"`, "", envs.RedactionPolicyNone},
		{`Name is will or Name has felix`, `name = "will" OR name ~ "felix"`, "", envs.RedactionPolicyNone}, // comparator aliases
		{`will or Name ~ "felix"`, `name ~ "will" OR name ~ "felix"`, "", envs.RedactionPolicyNone},

		// boolean operator precedence is AND before OR, even when AND is implicit
		{`will and felix or matt amber`, `(name ~ "will" AND name ~ "felix") OR (name ~ "matt" AND name ~ "amber")`, "", envs.RedactionPolicyNone},

		// boolean combinations can themselves be combined
		{
			`(Age < 18 and Gender = "male") or (Age > 18 and Gender = "female")`,
			`(age < 18 AND gender = "male") OR (age > 18 AND gender = "female")`,
			"",
			envs.RedactionPolicyNone,
		},

		{`xyz != ""`, "", "can't resolve 'xyz' to attribute, scheme or field", envs.RedactionPolicyNone},

		{`name = "O\"Leary"`, `name = "O\"Leary"`, "", envs.RedactionPolicyNone}, // string unquoting
	}

	fields := map[string]assets.Field{
		"age":    types.NewField(assets.FieldUUID("f1b5aea6-6586-41c7-9020-1a6326cc6565"), "age", "Age", assets.FieldTypeNumber),
		"gender": types.NewField(assets.FieldUUID("d66a7823-eada-40e5-9a3a-57239d4690bf"), "gender", "Gender", assets.FieldTypeText),
		"state":  types.NewField(assets.FieldUUID("165def68-3216-4ebf-96bc-f6f1ee5bd966"), "state", "State", assets.FieldTypeState),
		"dob":    types.NewField(assets.FieldUUID("85baf5e1-b57a-46dc-a726-a84e8c4229c7"), "dob", "DOB", assets.FieldTypeDatetime),
	}
	fieldResolver := func(key string) assets.Field { return fields[key] }

	for _, tc := range tests {
		parsed, err := contactql.ParseQuery(tc.text, tc.redact, fieldResolver)
		if tc.err != "" {
			assert.EqualError(t, err, tc.err, "error mismatch for '%s'", tc.text)
			assert.Nil(t, parsed)
		} else {
			assert.NoError(t, err, "unexpected error for '%s'", tc.text)
			assert.Equal(t, tc.parsed, parsed.String(), "parse mismatch for '%s'", tc.text)
		}
	}
}

func TestParsingErrors(t *testing.T) {
	_, err := contactql.ParseQuery("name = ", envs.RedactionPolicyNone, nil)
	assert.EqualError(t, err, "mismatched input '<EOF>' expecting {TEXT, STRING}")
}
