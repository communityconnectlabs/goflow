package dialogflowcl

import (
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/utils"
	"github.com/nyaruka/gocommon/httpx"
	"golang.org/x/text/language"
	dialogflowpb "google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
	"net/http"
)

// a classification service implementation for a Google dialogflow
type service struct {
	client     *Client
	classifier *flows.Classifier
	redactor   utils.Redactor
}

// NewService creates a new classification service
func NewService(httpClient *http.Client, httpRetries *httpx.RetryConfig, classifier *flows.Classifier, credentialJSON []byte, projectID string) flows.ClassificationService {
	return &service{
		client:     NewClient(httpClient, httpRetries, credentialJSON, projectID),
		classifier: classifier,
	}
}

func (s *service) Classify(session flows.Session, input string, logHTTP flows.HTTPLogCallback) (*flows.Classification, error) {
	languageCode := string(session.Runs()[0].Contact().Language())
	ISO1Tag, err := language.Parse(languageCode)
	if err != nil {
		return nil, err
	}

	response, trace, err := s.client.DetectIntentText(input, ISO1Tag.String())
	if trace != nil {
		logHTTP(flows.NewHTTPLog(trace, flows.HTTPStatusFromCode, s.redactor))
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

	return result, nil
}

func (s *service) DetectIntent(session flows.Session, input string) (*dialogflowpb.DetectIntentResponse, error) {
	languageCode := string(session.Runs()[0].Contact().Language())
	ISO1Tag, err := language.Parse(languageCode)
	if err != nil {
		return nil, err
	}

	return s.client.DetectIntent(input, ISO1Tag.String())
}

var _ flows.ClassificationService = (*service)(nil)
