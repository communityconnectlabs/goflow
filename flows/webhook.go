package flows

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/greatnonprofits-nfp/goflow/utils"

	"github.com/pkg/errors"
)

// LegacyWebhookPayload is a template that matches the JSON payload sent by legacy webhooks
var LegacyWebhookPayload = `@(json(object(
  "contact", object("uuid", contact.uuid, "name", contact.name, "urn", contact.urn),
  "flow", run.flow,
  "path", run.path,
  "results", foreach_value(results, extract_object, "category", "category_localized", "created_on", "input", "name", "node_uuid", "value"),
  "run", object("uuid", run.uuid, "created_on", run.created_on),
  "input", if(
    input,
    object(
      "attachments", foreach(input.attachments, attachment_parts),
      "channel", input.channel,
      "created_on", input.created_on,
      "text", input.text,
      "type", input.type,
      "urn", if(
        input.urn,
        object(
          "display", default(format_urn(input.urn), ""),
          "path", urn_parts(input.urn).path,
          "scheme", urn_parts(input.urn).scheme
        ),
        null
      ),
      "uuid", input.uuid
    ),
    null
  ),
  "channel", default(input.channel, null)
)))`

// response content-types that we'll fetch
var fetchResponseContentTypes = map[string]bool{
	"application/json":       true,
	"application/javascript": true,
	"application/xml":        true,
	"text/html":              true,
	"text/plain":             true,
	"text/xml":               true,
	"text/javascript":        true,
}

// WebhookStatus represents the status of a WebhookRequest
type WebhookStatus string

const (
	// WebhookStatusSuccess represents that the webhook was successful
	WebhookStatusSuccess WebhookStatus = "success"

	// WebhookStatusConnectionError represents that the webhook had a connection error
	WebhookStatusConnectionError WebhookStatus = "connection_error"

	// WebhookStatusResponseError represents that the webhook response had a non 2xx status code
	WebhookStatusResponseError WebhookStatus = "response_error"

	// WebhookStatusSubscriberGone represents a special state of resthook responses which indicate the caller must remove that subscriber
	WebhookStatusSubscriberGone WebhookStatus = "subscriber_gone"
)

// WebhookStatusFromCode determines the webhook status from the HTTP status code
func WebhookStatusFromCode(code int, isResthook bool) WebhookStatus {
	// https://zapier.com/developer/documentation/v2/rest-hooks/
	if isResthook && code == 410 {
		return WebhookStatusSubscriberGone
	}
	if code/100 == 2 {
		return WebhookStatusSuccess
	}
	return WebhookStatusResponseError
}

func (r WebhookStatus) String() string {
	return string(r)
}

// WebhookCall is a call made to an external service
type WebhookCall struct {
	url           string
	resthook      string
	request       *http.Request
	response      *http.Response
	status        WebhookStatus
	timeTaken     time.Duration
	requestTrace  string
	responseTrace string
	bodyIgnored   bool
}

// MakeWebhookCall fires the passed in http request, returning any errors encountered. RequestResponse is always set
// regardless of any errors being set
func MakeWebhookCall(session Session, request *http.Request, resthook string) (*WebhookCall, error) {
	var response *http.Response
	var requestDump string
	var err error
	var timeTaken time.Duration

	if session.Engine().DisableWebhooks() {
		response, requestDump, err = session.Engine().HTTPClient().MockWithDump(request, 200, "DISABLED")
	} else {
		start := utils.Now()
		response, requestDump, err = session.Engine().HTTPClient().DoWithDump(request)
		timeTaken = utils.Now().Sub(start)
	}

	if err != nil {
		return newWebhookCallFromError(request, requestDump, err), err
	}

	return newWebhookCallFromResponse(requestDump, response, session.Engine().MaxWebhookResponseBytes(), timeTaken, resthook)
}

// URL returns the full URL
func (w *WebhookCall) URL() string { return w.url }

// Resthook returns the resthook slug (if this call came from a resthook action)
func (w *WebhookCall) Resthook() string { return w.resthook }

// Method returns the full HTTP method
func (w *WebhookCall) Method() string { return w.request.Method }

// Status returns the response status message
func (w *WebhookCall) Status() WebhookStatus { return w.status }

// StatusCode returns the response status code
func (w *WebhookCall) StatusCode() int {
	if w.response != nil {
		return w.response.StatusCode
	}
	return 0
}

// TimeTaken returns the time taken to make the request
func (w *WebhookCall) TimeTaken() time.Duration { return w.timeTaken }

// Request returns the request trace
func (w *WebhookCall) Request() string { return w.requestTrace }

// Response returns the response trace
func (w *WebhookCall) Response() string { return w.responseTrace }

// BodyIgnored returns whether we ignored the body because we didn't recognize the content type
func (w *WebhookCall) BodyIgnored() bool {
	return w.bodyIgnored
}

// newWebhookCallFromError creates a new webhook call based on the passed in http request and error (when we received no response)
func newWebhookCallFromError(request *http.Request, requestTrace string, requestError error) *WebhookCall {
	return &WebhookCall{
		url:           request.URL.String(),
		request:       request,
		response:      nil,
		status:        WebhookStatusConnectionError,
		requestTrace:  requestTrace,
		responseTrace: requestError.Error(),
	}
}

// newWebhookCallFromResponse creates a new RequestResponse based on the passed in http Response
func newWebhookCallFromResponse(requestTrace string, response *http.Response, maxBodyBytes int, timeTaken time.Duration, resthook string) (*WebhookCall, error) {
	defer response.Body.Close()

	// save response trace without body which will be parsed separately
	responseTrace, err := httputil.DumpResponse(response, false)
	if err != nil {
		return nil, err
	}

	w := &WebhookCall{
		url:           response.Request.URL.String(),
		resthook:      resthook,
		request:       response.Request,
		response:      response,
		status:        WebhookStatusFromCode(response.StatusCode, resthook != ""),
		requestTrace:  requestTrace,
		responseTrace: string(responseTrace),
		timeTaken:     timeTaken,
	}

	// we will only read up to our max body bytes limit
	bodyReader := io.LimitReader(response.Body, int64(maxBodyBytes)+1)
	var bodySniffed []byte

	// hopefully we got a content-type header
	contentTypeHeader := response.Header.Get("Content-Type")
	contentType, _, _ := mime.ParseMediaType(contentTypeHeader)

	// but if not, read first 512 bytes to sniff the content-type
	if contentType == "" {
		bodySniffed = make([]byte, 512)
		bodyBytesRead, err := bodyReader.Read(bodySniffed)
		if err != nil && err != io.EOF {
			return nil, err
		}
		bodySniffed = bodySniffed[0:bodyBytesRead]

		contentType, _, _ = mime.ParseMediaType(http.DetectContentType(bodySniffed))
	}

	// only save response body's if we have a supported content-type
	saveBody := fetchResponseContentTypes[contentType]

	if saveBody {
		bodyBytes, err := ioutil.ReadAll(bodyReader)
		if err != nil {
			return nil, err
		}

		// if we have no remaining bytes, error because the body was too big
		if bodyReader.(*io.LimitedReader).N <= 0 {
			return nil, errors.Errorf("webhook response body exceeds %d bytes limit", maxBodyBytes)
		}

		if len(bodySniffed) > 0 {
			bodyBytes = append(bodySniffed, bodyBytes...)
		}

		w.responseTrace += string(bodyBytes)
	} else {
		w.bodyIgnored = true
	}

	return w, nil
}

// ExtractResponseBody extracts a JSON body from a webhook call response trace
func ExtractResponseBody(response string) json.RawMessage {
	parts := strings.SplitN(response, "\r\n\r\n", 2)

	// this response doesn't have a body
	if len(parts) != 2 || len(parts[1]) == 0 {
		return nil
	}

	body := []byte(parts[1])

	// check if body is valid JSON and can be returned as is
	if json.Valid(body) {
		return body
	}

	// if not, treat body as text and encode as a JSON string
	asString, _ := json.Marshal(string(body))
	return asString
}
