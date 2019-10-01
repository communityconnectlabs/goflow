package hints

import (
	"github.com/greatnonprofits-nfp/goflow/flows"
)

func init() {
	RegisterType(TypeVideo, func() flows.Hint { return &VideoHint{} })
}

// TypeVideo is the type of our video hint
const TypeVideo string = "video"

// VideoHint requests a message with an video attachment
type VideoHint struct {
	baseHint
}

// NewVideoHint creates a new video hint
func NewVideoHint() *VideoHint {
	return &VideoHint{
		baseHint: newBaseHint(TypeVideo),
	}
}
