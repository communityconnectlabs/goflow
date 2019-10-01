package flows_test

import (
	"fmt"
	"testing"

	"github.com/nyaruka/gocommon/urns"
	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/excellent/types"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/test"
	"github.com/greatnonprofits-nfp/goflow/utils"

	"github.com/stretchr/testify/assert"
)

func TestChannel(t *testing.T) {
	env := utils.NewEnvironmentBuilder().Build()

	utils.SetUUIDGenerator(test.NewSeededUUIDGenerator(1234))
	defer utils.SetUUIDGenerator(utils.DefaultUUIDGenerator)

	rolesDefault := []assets.ChannelRole{assets.ChannelRoleSend, assets.ChannelRoleReceive}
	ch := test.NewChannel("Android", "+250961111111", []string{"tel"}, rolesDefault, nil)

	assert.Equal(t, assets.ChannelUUID("c00e5d67-c275-4389-aded-7d8b151cbd5b"), ch.UUID())
	assert.Equal(t, "Android", ch.Name())
	assert.Equal(t, []string{"tel"}, ch.Schemes())
	assert.Equal(t, "+250961111111", ch.Address())
	assert.Equal(t, "+250961111111 (Android)", fmt.Sprintf("%s", ch))

	test.AssertXEqual(t, types.NewXObject(map[string]types.XValue{
		"__default__": types.NewXText("Android"),
		"uuid":        types.NewXText(string(ch.UUID())),
		"name":        types.NewXText("Android"),
		"address":     types.NewXText("+250961111111"),
	}), flows.Context(env, ch))

	assert.Equal(t, assets.NewChannelReference(ch.UUID(), "Android"), ch.Reference())
	assert.True(t, ch.HasRole(assets.ChannelRoleSend))
	assert.False(t, ch.HasRole(assets.ChannelRoleCall))
}

func TestChannelSetGetForURN(t *testing.T) {
	rolesSend := []assets.ChannelRole{assets.ChannelRoleSend}
	rolesDefault := []assets.ChannelRole{assets.ChannelRoleSend, assets.ChannelRoleReceive}

	claro := test.NewTelChannel("Claro", "+593971111111", rolesDefault, nil, "EC", nil)
	mtn := test.NewTelChannel("MTN", "+250782222222", rolesDefault, nil, "RW", nil)
	tigo := test.NewTelChannel("Tigo", "+250723333333", rolesDefault, nil, "RW", nil)
	twitter := test.NewChannel("Twitter", "nyaruka", []string{"twitter", "twitterid"}, rolesDefault, nil)
	all := flows.NewChannelAssets([]assets.Channel{claro.Asset(), mtn.Asset(), tigo.Asset(), twitter.Asset()})

	// nil if no channel
	emptySet := flows.NewChannelAssets(nil)
	assert.Nil(t, emptySet.GetForURN(flows.NewContactURN(urns.URN("tel:+12345678999"), nil), assets.ChannelRoleSend))

	// nil if no channel with correct scheme
	assert.Nil(t, all.GetForURN(flows.NewContactURN(urns.URN("mailto:rowan@foo.bar"), nil), assets.ChannelRoleSend))

	// if URN has a preferred channel, that is always used
	assert.Equal(t, tigo, all.GetForURN(flows.NewContactURN(urns.URN("tel:+250962222222"), tigo), assets.ChannelRoleSend))

	// if there's only one channel for that scheme, it's used
	assert.Equal(t, twitter, all.GetForURN(flows.NewContactURN(urns.URN("twitter:nyaruka2"), nil), assets.ChannelRoleSend))

	// if there's only one channel for that country, it's used
	assert.Equal(t, claro, all.GetForURN(flows.NewContactURN(urns.URN("tel:+593971234567"), nil), assets.ChannelRoleSend))

	// if there's multiple channels, one with longest number overlap wins
	assert.Equal(t, mtn, all.GetForURN(flows.NewContactURN(urns.URN("tel:+250781234567"), nil), assets.ChannelRoleSend))
	assert.Equal(t, tigo, all.GetForURN(flows.NewContactURN(urns.URN("tel:+250721234567"), nil), assets.ChannelRoleSend))

	// if there's no overlap, then last/newest channel wins
	assert.Equal(t, tigo, all.GetForURN(flows.NewContactURN(urns.URN("tel:+250962222222"), nil), assets.ChannelRoleSend))

	// channels can be delegates for other channels
	android := test.NewChannel("Android", "+250723333333", []string{"tel"}, rolesDefault, nil)
	bulk := test.NewChannel("Bulk Sender", "1234", []string{"tel"}, rolesSend, android.Reference())
	all = flows.NewChannelAssets([]assets.Channel{android.Asset(), bulk.Asset()})

	// delegate will always be used if it has the requested role
	assert.Equal(t, android, all.GetForURN(flows.NewContactURN(urns.URN("tel:+250721234567"), nil), assets.ChannelRoleReceive))
	assert.Equal(t, bulk, all.GetForURN(flows.NewContactURN(urns.URN("tel:+250721234567"), nil), assets.ChannelRoleSend))

	// matching prefixes can be explicitly set too
	short1 := test.NewTelChannel("Shortcode 1", "1234", rolesSend, nil, "RW", []string{"25078", "25077"})
	short2 := test.NewTelChannel("Shortcode 2", "1235", rolesSend, nil, "RW", []string{"25072"})
	all = flows.NewChannelAssets([]assets.Channel{short1.Asset(), short2.Asset()})

	assert.Equal(t, short1, all.GetForURN(flows.NewContactURN(urns.URN("tel:+250781234567"), nil), assets.ChannelRoleSend))
	assert.Equal(t, short1, all.GetForURN(flows.NewContactURN(urns.URN("tel:+250771234567"), nil), assets.ChannelRoleSend))
	assert.Equal(t, short2, all.GetForURN(flows.NewContactURN(urns.URN("tel:+250721234567"), nil), assets.ChannelRoleSend))
}
