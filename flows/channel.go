package flows

import (
	"fmt"
	"strings"

	"github.com/nyaruka/gocommon/urns"
	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/excellent/types"
	"github.com/greatnonprofits-nfp/goflow/utils"
)

// Channel represents a means for sending and receiving input during a flow run. It renders as its name in a template,
// and has the following properties which can be accessed:
//
//  * `uuid` the UUID of the channel
//  * `name` the name of the channel
//  * `address` the address of the channel
//
// Examples:
//
//   @contact.channel -> My Android Phone
//   @contact.channel.name -> My Android Phone
//   @contact.channel.address -> +12345671111
//   @input.channel.uuid -> 57f1078f-88aa-46f4-a59a-948a5739c03d
//   @(json(contact.channel)) -> {"address":"+12345671111","name":"My Android Phone","uuid":"57f1078f-88aa-46f4-a59a-948a5739c03d"}
//
// @context channel
type Channel struct {
	assets.Channel
}

func NewChannel(asset assets.Channel) *Channel {
	return &Channel{Channel: asset}
}

// Asset returns the underlying asset
func (c *Channel) Asset() assets.Channel { return c.Channel }

// Reference returns a reference to this channel
func (c *Channel) Reference() *assets.ChannelReference {
	if c == nil {
		return nil
	}
	return assets.NewChannelReference(c.UUID(), c.Name())
}

// SupportsScheme returns whether this channel supports the given URN scheme
func (c *Channel) SupportsScheme(scheme string) bool {
	for _, s := range c.Schemes() {
		if s == scheme {
			return true
		}
	}
	return false
}

// HasRole returns whether this channel has the given role
func (c *Channel) HasRole(role assets.ChannelRole) bool {
	for _, r := range c.Roles() {
		if r == role {
			return true
		}
	}
	return false
}

func (c *Channel) HasParent() bool {
	return c.Parent() != nil
}

// Context returns the properties available in expressions
func (c *Channel) Context(env utils.Environment) map[string]types.XValue {
	return map[string]types.XValue{
		"__default__": types.NewXText(c.Name()),
		"uuid":        types.NewXText(string(c.UUID())),
		"name":        types.NewXText(c.Name()),
		"address":     types.NewXText(c.Address()),
	}
}

func (c *Channel) String() string {
	return fmt.Sprintf("%s (%s)", c.Address(), c.Name())
}

// ChannelAssets provides access to all channel assets
type ChannelAssets struct {
	all    []*Channel
	byUUID map[assets.ChannelUUID]*Channel
}

// NewChannelAssets creates a new set of channel assets
func NewChannelAssets(channels []assets.Channel) *ChannelAssets {
	s := &ChannelAssets{
		all:    make([]*Channel, len(channels)),
		byUUID: make(map[assets.ChannelUUID]*Channel, len(channels)),
	}
	for i, asset := range channels {
		channel := NewChannel(asset)
		s.all[i] = channel
		s.byUUID[channel.UUID()] = channel
	}
	return s
}

// Get returns the channel with the given UUID
func (s *ChannelAssets) Get(uuid assets.ChannelUUID) *Channel {
	return s.byUUID[uuid]
}

// GetForURN returns the best channel for the given URN
func (s *ChannelAssets) GetForURN(urn *ContactURN, role assets.ChannelRole) *Channel {
	// if caller has told us which channel to use for this URN, use that
	if urn.Channel() != nil {
		return s.getDelegate(urn.Channel(), role)
	}

	// tel is a special case because we do number based matching
	if urn.URN().Scheme() == urns.TelScheme {
		countryCode := utils.DeriveCountryFromTel(urn.URN().Path())
		candidates := make([]*Channel, 0)

		for _, ch := range s.all {
			if ch.HasRole(role) && ch.SupportsScheme(urns.TelScheme) && (countryCode == "" || countryCode == ch.Country()) && !ch.HasParent() {
				candidates = append(candidates, ch)
			}
		}

		var channel *Channel
		if len(candidates) > 1 {
			// we don't have a channel for this contact yet, let's try to pick one from the same carrier
			// we need at least one digit to overlap to infer a channel
			contactNumber := strings.TrimPrefix(urn.URN().Path(), "+")
			maxOverlap := 0
			for _, candidate := range candidates {
				candidatePrefixes := candidate.MatchPrefixes()
				if len(candidatePrefixes) == 0 {
					candidatePrefixes = []string{strings.TrimPrefix(candidate.Address(), "+")}
				}

				for _, prefix := range candidatePrefixes {
					overlap := utils.PrefixOverlap(prefix, contactNumber)
					if overlap >= maxOverlap {
						maxOverlap = overlap
						channel = candidate
					}
				}
			}

		} else if len(candidates) == 1 {
			channel = candidates[0]
		}

		if channel != nil {
			return s.getDelegate(channel, role)
		}
	}

	return s.getForSchemeAndRole(urn.URN().Scheme(), role)
}

func (s *ChannelAssets) getForSchemeAndRole(scheme string, role assets.ChannelRole) *Channel {
	for _, ch := range s.all {
		if ch.HasRole(role) && ch.SupportsScheme(scheme) {
			return s.getDelegate(ch, role)
		}
	}
	return nil
}

// looks for a delegate for the given channel and defaults to the channel itself
func (s *ChannelAssets) getDelegate(channel *Channel, role assets.ChannelRole) *Channel {
	for _, ch := range s.all {
		if ch.HasParent() && ch.Parent().UUID == channel.UUID() && ch.HasRole(role) {
			return ch
		}
	}
	return channel
}
