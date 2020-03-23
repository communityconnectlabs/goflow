package httpx

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/greatnonprofits-nfp/goflow/utils/jsonx"
	"github.com/pkg/errors"
)

// MockRequestor is a requestor which can be mocked with responses for given URLs
type MockRequestor struct {
	mocks map[string][]MockResponse
}

// NewMockRequestor creates a new mock requestor with the given mocks
func NewMockRequestor(mocks map[string][]MockResponse) *MockRequestor {
	return &MockRequestor{mocks: mocks}
}

// Do returns the mocked reponse for the given request
func (r *MockRequestor) Do(client *http.Client, request *http.Request) (*http.Response, error) {
	url := request.URL.String()
	mockedResponses := r.mocks[url]
	if len(mockedResponses) == 0 {
		panic(fmt.Sprintf("missing mock for URL %s", url))
	}

	// pop the next mocked response for this URL
	mocked := mockedResponses[0]
	r.mocks[url] = mockedResponses[1:]

	if mocked.Status == 0 {
		return nil, errors.New("unable to connect to server")
	}

	return mocked.Make(request), nil
}

// HasUnused returns true if there are unused mocks leftover
func (r *MockRequestor) HasUnused() bool {
	for _, mocks := range r.mocks {
		if len(mocks) > 0 {
			return true
		}
	}
	return false
}

// Clone returns a clone of this requestor
func (r *MockRequestor) Clone() *MockRequestor {
	cloned := make(map[string][]MockResponse)
	for url, ms := range r.mocks {
		cloned[url] = ms
	}
	return NewMockRequestor(cloned)
}

func (r *MockRequestor) MarshalJSON() ([]byte, error) {
	return jsonx.Marshal(&r.mocks)
}

func (r *MockRequestor) UnmarshalJSON(data []byte) error {
	return jsonx.Unmarshal(data, &r.mocks)
}

var _ Requestor = (*MockRequestor)(nil)

type MockResponse struct {
	Status     int               `json:"status" validate:"required"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body" validate:"required"`
	BodyRepeat int               `json:"body_repeat,omitempty"`
}

// Make mocks making the given request and returning this as the response
func (m MockResponse) Make(request *http.Request) *http.Response {
	header := make(http.Header, len(m.Headers))
	for k, v := range m.Headers {
		header.Set(k, v)
	}

	body := m.Body
	if m.BodyRepeat > 1 {
		body = strings.Repeat(body, m.BodyRepeat)
	}

	return &http.Response{
		Request:       request,
		Status:        fmt.Sprintf("%d %s", m.Status, http.StatusText(m.Status)),
		StatusCode:    m.Status,
		Proto:         "HTTP/1.0",
		ProtoMajor:    1,
		ProtoMinor:    0,
		Header:        header,
		Body:          ioutil.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}

// MockConnectionError mocks a connection error
var MockConnectionError = MockResponse{0, nil, "", 0}

// NewMockResponse creates a new mock response
func NewMockResponse(status int, headers map[string]string, body string, bodyRepeat int) MockResponse {
	return MockResponse{status, headers, body, bodyRepeat}
}
