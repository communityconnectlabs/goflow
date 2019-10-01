package test

import (
	"io/ioutil"

	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/assets/static"
	"github.com/greatnonprofits-nfp/goflow/assets/static/types"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/engine"
	"github.com/greatnonprofits-nfp/goflow/utils"
)

// LoadSessionAssets loads a session assets instance from a static JSON file
func LoadSessionAssets(path string) (flows.SessionAssets, error) {
	assetsJSON, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	source, err := static.NewSource(assetsJSON)
	if err != nil {
		return nil, err
	}

	return engine.NewSessionAssets(source)
}

func LoadFlowFromAssets(path string, uuid assets.FlowUUID) (flows.Flow, error) {
	sa, err := LoadSessionAssets(path)
	if err != nil {
		return nil, err
	}

	return sa.Flows().Get(uuid)
}

func NewField(key string, name string, valueType assets.FieldType) *flows.Field {
	return flows.NewField(types.NewField(key, name, valueType))
}

func NewGroup(name string, query string) *flows.Group {
	return flows.NewGroup(types.NewGroup(assets.GroupUUID(utils.NewUUID()), name, query))
}

func NewChannel(name string, address string, schemes []string, roles []assets.ChannelRole, parent *assets.ChannelReference) *flows.Channel {
	return flows.NewChannel(types.NewChannel(assets.ChannelUUID(utils.NewUUID()), name, address, schemes, roles, parent))
}

func NewTelChannel(name string, address string, roles []assets.ChannelRole, parent *assets.ChannelReference, country string, matchPrefixes []string) *flows.Channel {
	return flows.NewChannel(types.NewTelChannel(assets.ChannelUUID(utils.NewUUID()), name, address, roles, parent, country, matchPrefixes))
}
