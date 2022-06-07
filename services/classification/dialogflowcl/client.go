package dialogflowcl

import (
	dialogflow "cloud.google.com/go/dialogflow/apiv2"
	"context"
	"fmt"
	"github.com/nyaruka/gocommon/httpx"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"google.golang.org/api/option"
	dialogflowpb "google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
	"net/http"
)

// IntentMatch is possible intent match
type IntentMatch struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	FulfillmentText string          `json:"text"`
	Confidence      decimal.Decimal `json:"confidence"`
}

type EntityMatch struct {
	Value      string          `json:"value"`
	Entity     string          `json:"entity"`
	Confidence decimal.Decimal `json:"confidence"`
}

// MessageResponse is the response from a /message request
type MessageResponse struct {
	Text     string                   `json:"text"`
	Intents  []IntentMatch            `json:"intents" validate:"required"`
	Entities map[string][]EntityMatch `json:"entities"`
}

func newMessageResponse(text string, intent IntentMatch) *MessageResponse {
	intents := []IntentMatch{intent}
	return &MessageResponse{
		Text:    text,
		Intents: intents,
	}
}

// Client is a basic dialogflow client
type Client struct {
	httpClient     *http.Client
	httpRetries    *httpx.RetryConfig
	CredentialJSON []byte
	ProjectID      string
}

// NewClient creates a new client
func NewClient(httpClient *http.Client, httpRetries *httpx.RetryConfig, credentialJSON []byte, projectID string) *Client {
	return &Client{
		httpClient:     httpClient,
		httpRetries:    httpRetries,
		CredentialJSON: credentialJSON,
		ProjectID:      projectID,
	}
}

func (c Client) DetectIntentText(q, languageCode, contactId string) (*MessageResponse, *httpx.Trace, error) {
	response, err := c.DetectIntent(q, languageCode, contactId)
	if err != nil {
		return nil, nil, err
	}

	queryResult := response.GetQueryResult()
	text := queryResult.GetFulfillmentText()
	responseIntent := queryResult.Intent

	intent := IntentMatch{
		ID:              responseIntent.GetName(),
		Name:            responseIntent.GetDisplayName(),
		Confidence:      decimal.NewFromFloat32(queryResult.GetIntentDetectionConfidence()),
		FulfillmentText: text,
	}
	msgResponse := newMessageResponse(text, intent)
	return msgResponse, nil, nil
}

func (c Client) DetectIntent(q, languageCode, contactId string) (*dialogflowpb.DetectIntentResponse, error) {
	projectID := c.ProjectID
	sessionID := fmt.Sprintf("%s%s", projectID, contactId)
	ctx := context.Background()
	sessionClient, err := dialogflow.NewSessionsClient(ctx, option.WithCredentialsJSON(c.CredentialJSON))
	if err != nil {
		return nil, err
	}
	defer sessionClient.Close()

	if projectID == "" {
		return nil, errors.New(fmt.Sprintf("Received empty project (%s)", projectID))
	}

	sessionPath := fmt.Sprintf("projects/%s/agent/sessions/%s", projectID, sessionID)
	textInput := dialogflowpb.TextInput{Text: q, LanguageCode: languageCode}
	queryTextInput := dialogflowpb.QueryInput_Text{Text: &textInput}
	queryInput := dialogflowpb.QueryInput{Input: &queryTextInput}
	request := &dialogflowpb.DetectIntentRequest{Session: sessionPath, QueryInput: &queryInput}
	return sessionClient.DetectIntent(ctx, request)
}
