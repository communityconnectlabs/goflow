package flows

import (
	"github.com/nyaruka/gocommon/urns"
	"github.com/nyaruka/goflow/assets"
)


type FeedbackRequest struct {
	URN_                urns.URN                 `json:"urn,omitempty" validate:"omitempty,urn"`
	Channel_            *assets.ChannelReference `json:"channel,omitempty"`
	StarRatingQuestion_ string                   `json:"star_rating_question,omitempty"`
	CommentQuestion_    string                   `json:"comment_question,omitempty"`
}

// URN returns the URN of this message
func (fr *FeedbackRequest) URN() urns.URN { return fr.URN_ }

// SetURN returns the URN of this message
func (fr *FeedbackRequest) SetURN(urn urns.URN) { fr.URN_ = urn }

// Channel returns the channel of this message
func (fr *FeedbackRequest) Channel() *assets.ChannelReference { return fr.Channel_ }

// Channel returns the channel of this message
func (fr *FeedbackRequest) StarRatingQuestion() string { return fr.StarRatingQuestion_ }

// Channel returns the channel of this message
func (fr *FeedbackRequest) CommentQuestion() string { return fr.CommentQuestion_ }


// NewFeedbackRequest creates a new feedback request
func NewFeedbackRequest(urn urns.URN, channel *assets.ChannelReference, starRatingQuestion string, commentQuestion string) *FeedbackRequest {
	return &FeedbackRequest{
		URN_: urn,
		Channel_: channel,
		StarRatingQuestion_: starRatingQuestion,
		CommentQuestion_: commentQuestion,
	}
}
