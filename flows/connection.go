package flows

import (
	"github.com/nyaruka/gocommon/jsonx"
	"github.com/nyaruka/gocommon/urns"
	"github.com/greatnonprofits-nfp/goflow/assets"
)

// Connection represents a connection to a specific channel using a specific URN
type Connection struct {
	channel           *assets.ChannelReference
	urn               urns.URN
	externalID        string
	twilioCredentials string
}

// NewConnection creates a new connection
func NewConnection(channel *assets.ChannelReference, urn urns.URN, externalID string, twilioCredentials string) *Connection {
	return &Connection{channel: channel, urn: urn, externalID: externalID, twilioCredentials: twilioCredentials}
}

// Channel returns a reference to the channel
func (c *Connection) Channel() *assets.ChannelReference { return c.channel }

// URN returns the URN
func (c *Connection) URN() urns.URN { return c.urn }

// ExternalID returns the External ID of this channel connection
func (c *Connection) ExternalID() string { return c.externalID }

// TwilioCredentials returns the External ID of this channel connection
func (c *Connection) TwilioCredentials() string { return c.twilioCredentials }

//------------------------------------------------------------------------------------------
// JSON Encoding / Decoding
//------------------------------------------------------------------------------------------

type connectionEnvelope struct {
	Channel           *assets.ChannelReference `json:"channel" validate:"required,dive"`
	URN               urns.URN                 `json:"urn" validate:"required,urn"`
	ExternalID        string                   `json:"external_id"`
	TwilioCredentials string                   `json:"twilio_credentials"`
}

// UnmarshalJSON unmarshals a connection from JSON
func (c *Connection) UnmarshalJSON(data []byte) error {
	e := &connectionEnvelope{}
	if err := jsonx.Unmarshal(data, e); err != nil {
		return err
	}

	c.channel = e.Channel
	c.urn = e.URN
	c.externalID = e.ExternalID
	c.twilioCredentials = e.TwilioCredentials
	return nil
}

// MarshalJSON marshals this connection into JSON
func (c *Connection) MarshalJSON() ([]byte, error) {
	return jsonx.Marshal(&connectionEnvelope{
		Channel:    c.channel,
		URN:        c.urn,
		ExternalID: c.externalID,
		TwilioCredentials: c.twilioCredentials,
	})
}
