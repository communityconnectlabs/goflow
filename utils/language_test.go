package utils_test

import (
	"testing"

	"github.com/greatnonprofits-nfp/goflow/utils"

	"github.com/stretchr/testify/assert"
)

func TestLanguage(t *testing.T) {
	lang, err := utils.ParseLanguage("ENG")
	assert.NoError(t, err)
	assert.Equal(t, utils.Language("eng"), lang)

	_, err = utils.ParseLanguage("base")
	assert.EqualError(t, err, "iso-639-3 codes must be 3 characters, got: base")

	_, err = utils.ParseLanguage("xzx")
	assert.EqualError(t, err, "unrecognized language code: xzx")
}
