package wit

import (
	"net/http"

	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/utils/httpx"
	"strings"
)

// a classification service implementation for a wit.ai app
type service struct {
	httpClient  *http.Client
	httpRetries *httpx.RetryConfig
	classifier  *flows.Classifier
	accessToken string
}

// NewService creates a new classification service
func NewService(httpClient *http.Client, httpRetries *httpx.RetryConfig, classifier *flows.Classifier, accessToken string) flows.ClassificationService {
	return &service{
		httpClient:  httpClient,
		httpRetries: httpRetries,
		classifier:  classifier,
		accessToken: accessToken,
	}
}

func (s *service) Classify(session flows.Session, input string, logHTTP flows.HTTPLogCallback) (*flows.Classification, error) {
	client := NewClient(s.httpClient, s.httpRetries, s.accessToken)

	response, trace, err := client.Message(input)
	if trace != nil {
		logHTTP(flows.NewHTTPLog(trace, flows.HTTPStatusFromCode))
	}
	if err != nil {
		return nil, err
	}

	result := &flows.Classification{
		Intents:  make([]flows.ExtractedIntent, len(response.Intents)),
		Entities: make(map[string][]flows.ExtractedEntity),
	}

	for i, intent := range response.Intents {
		result.Intents[i] = flows.ExtractedIntent{Name: intent.Name, Confidence: intent.Confidence}
	}

	for nameAndRole, entity := range response.Entities {
		name := strings.Split(nameAndRole, ":")[0]
		entities := make([]flows.ExtractedEntity, 0, len(entity))
		for _, candidate := range entity {
			entities = append(entities, flows.ExtractedEntity{
				Value:      candidate.Value,
				Confidence: candidate.Confidence,
			})
		}
		result.Entities[name] = entities
	}

	return result, nil
}

var _ flows.ClassificationService = (*service)(nil)
