package utils_test

import (
	"testing"

	"github.com/greatnonprofits-nfp/goflow/utils"

	"github.com/stretchr/testify/assert"
)

func TestIsNil(t *testing.T) {
	assert.True(t, utils.IsNil(nil))
	assert.True(t, utils.IsNil(error(nil)))
	assert.False(t, utils.IsNil(""))
}

func TestMaxInt(t *testing.T) {
	assert.Equal(t, 1, utils.MaxInt(0, 1))
	assert.Equal(t, 1, utils.MaxInt(1, 0))
	assert.Equal(t, 1, utils.MaxInt(1, -1))
}

func TestMinInt(t *testing.T) {
	assert.Equal(t, 0, utils.MinInt(0, 1))
	assert.Equal(t, 0, utils.MinInt(1, 0))
	assert.Equal(t, -1, utils.MinInt(1, -1))
}

func TestDeriveCountryFromTel(t *testing.T) {
	assert.Equal(t, "RW", utils.DeriveCountryFromTel("+250788383383"))
	assert.Equal(t, "EC", utils.DeriveCountryFromTel("+593979000000"))
	assert.Equal(t, "", utils.DeriveCountryFromTel("1234"))
}
