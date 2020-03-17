package bothub

import (
	"github.com/greatnonprofits-nfp/goflow/flows"
)

// a classification service implementation for a bothub.it bot
type service struct {
	classifier  *flows.Classifier
	accessToken string
}

// NewService creates a new classification service
func NewService(classifier *flows.Classifier, accessToken string) flows.ClassificationService {
	return &service{
		classifier:  classifier,
		accessToken: accessToken,
	}
}

func (s *service) Classify(session flows.Session, input string, logHTTP flows.HTTPLogCallback) (*flows.Classification, error) {
	client := NewClient(session.Engine().HTTPClient(), s.accessToken)

	response, trace, err := client.Parse(input)
	if trace != nil {
		logHTTP(flows.NewHTTPLog(trace, flows.HTTPStatusFromCode))
	}
	if err != nil {
		return nil, err
	}

	result := &flows.Classification{
		Intents:  make([]flows.ExtractedIntent, len(response.IntentRanking)),
		Entities: make(map[string][]flows.ExtractedEntity, len(response.LabelsList)),
	}

	for i, intent := range response.IntentRanking {
		result.Intents[i] = flows.ExtractedIntent{Name: intent.Name, Confidence: intent.Confidence}
	}

	for label, entities := range response.Entities {
		result.Entities[label] = make([]flows.ExtractedEntity, 0, len(response.Entities))

		for _, entity := range entities {
			result.Entities[label] = append(result.Entities[label], flows.ExtractedEntity{Value: entity.Entity, Confidence: entity.Confidence})
		}
	}

	return result, nil
}

var _ flows.ClassificationService = (*service)(nil)
