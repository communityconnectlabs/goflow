package definition

import (
	"sync"

	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/flows"
)

// implemention of FlowAssets which provides lazy loading and validation of flows
type flowAssets struct {
	byUUID map[assets.FlowUUID]flows.Flow

	mutex  sync.Mutex
	source assets.AssetSource
}

// NewFlowAssets creates a new flow assets
func NewFlowAssets(source assets.AssetSource) flows.FlowAssets {
	return &flowAssets{
		byUUID: make(map[assets.FlowUUID]flows.Flow),
		source: source,
	}
}

// Get returns the flow with the given UUID
func (a *flowAssets) Get(uuid assets.FlowUUID) (flows.Flow, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	flow := a.byUUID[uuid]
	if flow != nil {
		return flow, nil
	}

	asset, err := a.source.Flow(uuid)
	if err != nil {
		return nil, err
	}

	flow, err = ReadFlow(asset.Definition())
	if err != nil {
		return nil, err
	}

	a.byUUID[flow.UUID()] = flow
	return flow, nil
}
