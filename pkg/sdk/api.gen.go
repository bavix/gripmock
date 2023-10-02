// Package sdk provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.14.0 DO NOT EDIT.
package sdk

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/oapi-codegen/runtime"
	openapi_types "github.com/oapi-codegen/runtime/types"
	codes "google.golang.org/grpc/codes"

	"github.com/bavix/gripmock/pkg/json"
)

// ID defines model for ID.
type ID = openapi_types.UUID

// ListID defines model for ListID.
type ListID = []ID

// MessageOK defines model for MessageOK.
type MessageOK struct {
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

// SearchRequest defines model for SearchRequest.
type SearchRequest struct {
	Data    interface{}       `json:"data"`
	Headers map[string]string `json:"headers,omitempty"`
	Id      *ID               `json:"id,omitempty"`
	Method  string            `json:"method"`
	Service string            `json:"service"`
}

// SearchResponse defines model for SearchResponse.
type SearchResponse struct {
	Code    *codes.Code       `json:"code,omitempty"`
	Data    interface{}       `json:"data"`
	Error   string            `json:"error"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Stub defines model for Stub.
type Stub struct {
	Headers StubHeaders `json:"headers,omitempty"`
	Id      *ID         `json:"id,omitempty"`
	Input   StubInput   `json:"input"`
	Method  string      `json:"method"`
	Output  StubOutput  `json:"output"`
	Service string      `json:"service"`
}

// StubHeaders defines model for StubHeaders.
type StubHeaders struct {
	Contains map[string]string `json:"contains,omitempty"`
	Equals   map[string]string `json:"equals,omitempty"`
	Matches  map[string]string `json:"matches,omitempty"`
}

// StubInput defines model for StubInput.
type StubInput struct {
	Contains map[string]interface{} `json:"contains,omitempty"`
	Equals   map[string]interface{} `json:"equals,omitempty"`
	Matches  map[string]interface{} `json:"matches,omitempty"`
}

// StubList defines model for StubList.
type StubList = []Stub

// StubOutput defines model for StubOutput.
type StubOutput struct {
	Code    *codes.Code            `json:"code,omitempty"`
	Data    map[string]interface{} `json:"data"`
	Error   string                 `json:"error"`
	Headers map[string]string      `json:"headers,omitempty"`
}

// AddStubJSONBody defines parameters for AddStub.
type AddStubJSONBody struct {
	union json.RawMessage
}

// AddStubJSONRequestBody defines body for AddStub for application/json ContentType.
type AddStubJSONRequestBody AddStubJSONBody

// BatchStubsDeleteJSONRequestBody defines body for BatchStubsDelete for application/json ContentType.
type BatchStubsDeleteJSONRequestBody = ListID

// SearchStubsJSONRequestBody defines body for SearchStubs for application/json ContentType.
type SearchStubsJSONRequestBody = SearchRequest

// RequestEditorFn  is the function signature for the RequestEditor callback function
type RequestEditorFn func(ctx context.Context, req *http.Request) error

// Doer performs HTTP requests.
//
// The standard http.Client implements this interface.
type HttpRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client which conforms to the OpenAPI3 specification for this service.
type Client struct {
	// The endpoint of the server conforming to this interface, with scheme,
	// https://api.deepmap.com for example. This can contain a path relative
	// to the server, such as https://api.deepmap.com/dev-test, and all the
	// paths in the swagger spec will be appended to the server.
	Server string

	// Doer for performing requests, typically a *http.Client with any
	// customized settings, such as certificate chains.
	Client HttpRequestDoer

	// A list of callbacks for modifying requests which are generated before sending over
	// the network.
	RequestEditors []RequestEditorFn
}

// ClientOption allows setting custom parameters during construction
type ClientOption func(*Client) error

// Creates a new Client, with reasonable defaults
func NewClient(server string, opts ...ClientOption) (*Client, error) {
	// create a client with sane default values
	client := Client{
		Server: server,
	}
	// mutate client and add all optional params
	for _, o := range opts {
		if err := o(&client); err != nil {
			return nil, err
		}
	}
	// ensure the server URL always has a trailing slash
	if !strings.HasSuffix(client.Server, "/") {
		client.Server += "/"
	}
	// create httpClient, if not already present
	if client.Client == nil {
		client.Client = &http.Client{}
	}
	return &client, nil
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HttpRequestDoer) ClientOption {
	return func(c *Client) error {
		c.Client = doer
		return nil
	}
}

// WithRequestEditorFn allows setting up a callback function, which will be
// called right before sending the request. This can be used to mutate the request.
func WithRequestEditorFn(fn RequestEditorFn) ClientOption {
	return func(c *Client) error {
		c.RequestEditors = append(c.RequestEditors, fn)
		return nil
	}
}

// The interface specification for the client above.
type ClientInterface interface {
	// Liveness request
	Liveness(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// Readiness request
	Readiness(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// PurgeStubs request
	PurgeStubs(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// ListStubs request
	ListStubs(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// AddStubWithBody request with any body
	AddStubWithBody(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error)

	AddStub(ctx context.Context, body AddStubJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error)

	// BatchStubsDeleteWithBody request with any body
	BatchStubsDeleteWithBody(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error)

	BatchStubsDelete(ctx context.Context, body BatchStubsDeleteJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error)

	// SearchStubsWithBody request with any body
	SearchStubsWithBody(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error)

	SearchStubs(ctx context.Context, body SearchStubsJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error)

	// ListUnusedStubs request
	ListUnusedStubs(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// DeleteStubByID request
	DeleteStubByID(ctx context.Context, uuid ID, reqEditors ...RequestEditorFn) (*http.Response, error)
}

func (c *Client) Liveness(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewLivenessRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) Readiness(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewReadinessRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) PurgeStubs(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewPurgeStubsRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) ListStubs(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewListStubsRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) AddStubWithBody(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewAddStubRequestWithBody(c.Server, contentType, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) AddStub(ctx context.Context, body AddStubJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewAddStubRequest(c.Server, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) BatchStubsDeleteWithBody(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewBatchStubsDeleteRequestWithBody(c.Server, contentType, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) BatchStubsDelete(ctx context.Context, body BatchStubsDeleteJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewBatchStubsDeleteRequest(c.Server, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) SearchStubsWithBody(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewSearchStubsRequestWithBody(c.Server, contentType, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) SearchStubs(ctx context.Context, body SearchStubsJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewSearchStubsRequest(c.Server, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) ListUnusedStubs(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewListUnusedStubsRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) DeleteStubByID(ctx context.Context, uuid ID, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewDeleteStubByIDRequest(c.Server, uuid)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// NewLivenessRequest generates requests for Liveness
func NewLivenessRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/health/liveness")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewReadinessRequest generates requests for Readiness
func NewReadinessRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/health/readiness")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewPurgeStubsRequest generates requests for PurgeStubs
func NewPurgeStubsRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/stubs")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("DELETE", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewListStubsRequest generates requests for ListStubs
func NewListStubsRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/stubs")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewAddStubRequest calls the generic AddStub builder with application/json body
func NewAddStubRequest(server string, body AddStubJSONRequestBody) (*http.Request, error) {
	var bodyReader io.Reader
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)
	return NewAddStubRequestWithBody(server, "application/json", bodyReader)
}

// NewAddStubRequestWithBody generates requests for AddStub with any type of body
func NewAddStubRequestWithBody(server string, contentType string, body io.Reader) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/stubs")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", queryURL.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", contentType)

	return req, nil
}

// NewBatchStubsDeleteRequest calls the generic BatchStubsDelete builder with application/json body
func NewBatchStubsDeleteRequest(server string, body BatchStubsDeleteJSONRequestBody) (*http.Request, error) {
	var bodyReader io.Reader
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)
	return NewBatchStubsDeleteRequestWithBody(server, "application/json", bodyReader)
}

// NewBatchStubsDeleteRequestWithBody generates requests for BatchStubsDelete with any type of body
func NewBatchStubsDeleteRequestWithBody(server string, contentType string, body io.Reader) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/stubs/batchDelete")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", queryURL.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", contentType)

	return req, nil
}

// NewSearchStubsRequest calls the generic SearchStubs builder with application/json body
func NewSearchStubsRequest(server string, body SearchStubsJSONRequestBody) (*http.Request, error) {
	var bodyReader io.Reader
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)
	return NewSearchStubsRequestWithBody(server, "application/json", bodyReader)
}

// NewSearchStubsRequestWithBody generates requests for SearchStubs with any type of body
func NewSearchStubsRequestWithBody(server string, contentType string, body io.Reader) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/stubs/search")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", queryURL.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", contentType)

	return req, nil
}

// NewListUnusedStubsRequest generates requests for ListUnusedStubs
func NewListUnusedStubsRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/stubs/unused")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewDeleteStubByIDRequest generates requests for DeleteStubByID
func NewDeleteStubByIDRequest(server string, uuid ID) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "uuid", runtime.ParamLocationPath, uuid)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/stubs/%s", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("DELETE", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (c *Client) applyEditors(ctx context.Context, req *http.Request, additionalEditors []RequestEditorFn) error {
	for _, r := range c.RequestEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	for _, r := range additionalEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ClientWithResponses builds on ClientInterface to offer response payloads
type ClientWithResponses struct {
	ClientInterface
}

// NewClientWithResponses creates a new ClientWithResponses, which wraps
// Client with return type handling
func NewClientWithResponses(server string, opts ...ClientOption) (*ClientWithResponses, error) {
	client, err := NewClient(server, opts...)
	if err != nil {
		return nil, err
	}
	return &ClientWithResponses{client}, nil
}

// WithBaseURL overrides the baseURL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		newBaseURL, err := url.Parse(baseURL)
		if err != nil {
			return err
		}
		c.Server = newBaseURL.String()
		return nil
	}
}

// ClientWithResponsesInterface is the interface specification for the client with responses above.
type ClientWithResponsesInterface interface {
	// LivenessWithResponse request
	LivenessWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*LivenessResponse, error)

	// ReadinessWithResponse request
	ReadinessWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*ReadinessResponse, error)

	// PurgeStubsWithResponse request
	PurgeStubsWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*PurgeStubsResponse, error)

	// ListStubsWithResponse request
	ListStubsWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*ListStubsResponse, error)

	// AddStubWithBodyWithResponse request with any body
	AddStubWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*AddStubResponse, error)

	AddStubWithResponse(ctx context.Context, body AddStubJSONRequestBody, reqEditors ...RequestEditorFn) (*AddStubResponse, error)

	// BatchStubsDeleteWithBodyWithResponse request with any body
	BatchStubsDeleteWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*BatchStubsDeleteResponse, error)

	BatchStubsDeleteWithResponse(ctx context.Context, body BatchStubsDeleteJSONRequestBody, reqEditors ...RequestEditorFn) (*BatchStubsDeleteResponse, error)

	// SearchStubsWithBodyWithResponse request with any body
	SearchStubsWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*SearchStubsResponse, error)

	SearchStubsWithResponse(ctx context.Context, body SearchStubsJSONRequestBody, reqEditors ...RequestEditorFn) (*SearchStubsResponse, error)

	// ListUnusedStubsWithResponse request
	ListUnusedStubsWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*ListUnusedStubsResponse, error)

	// DeleteStubByIDWithResponse request
	DeleteStubByIDWithResponse(ctx context.Context, uuid ID, reqEditors ...RequestEditorFn) (*DeleteStubByIDResponse, error)
}

type LivenessResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *MessageOK
}

// Status returns HTTPResponse.Status
func (r LivenessResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r LivenessResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type ReadinessResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *MessageOK
}

// Status returns HTTPResponse.Status
func (r ReadinessResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r ReadinessResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type PurgeStubsResponse struct {
	Body         []byte
	HTTPResponse *http.Response
}

// Status returns HTTPResponse.Status
func (r PurgeStubsResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r PurgeStubsResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type ListStubsResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *StubList
}

// Status returns HTTPResponse.Status
func (r ListStubsResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r ListStubsResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type AddStubResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *struct {
		union json.RawMessage
	}
}

// Status returns HTTPResponse.Status
func (r AddStubResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r AddStubResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type BatchStubsDeleteResponse struct {
	Body         []byte
	HTTPResponse *http.Response
}

// Status returns HTTPResponse.Status
func (r BatchStubsDeleteResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r BatchStubsDeleteResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type SearchStubsResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *SearchResponse
}

// Status returns HTTPResponse.Status
func (r SearchStubsResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r SearchStubsResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type ListUnusedStubsResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *StubList
}

// Status returns HTTPResponse.Status
func (r ListUnusedStubsResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r ListUnusedStubsResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type DeleteStubByIDResponse struct {
	Body         []byte
	HTTPResponse *http.Response
}

// Status returns HTTPResponse.Status
func (r DeleteStubByIDResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r DeleteStubByIDResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

// LivenessWithResponse request returning *LivenessResponse
func (c *ClientWithResponses) LivenessWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*LivenessResponse, error) {
	rsp, err := c.Liveness(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseLivenessResponse(rsp)
}

// ReadinessWithResponse request returning *ReadinessResponse
func (c *ClientWithResponses) ReadinessWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*ReadinessResponse, error) {
	rsp, err := c.Readiness(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseReadinessResponse(rsp)
}

// PurgeStubsWithResponse request returning *PurgeStubsResponse
func (c *ClientWithResponses) PurgeStubsWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*PurgeStubsResponse, error) {
	rsp, err := c.PurgeStubs(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParsePurgeStubsResponse(rsp)
}

// ListStubsWithResponse request returning *ListStubsResponse
func (c *ClientWithResponses) ListStubsWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*ListStubsResponse, error) {
	rsp, err := c.ListStubs(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseListStubsResponse(rsp)
}

// AddStubWithBodyWithResponse request with arbitrary body returning *AddStubResponse
func (c *ClientWithResponses) AddStubWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*AddStubResponse, error) {
	rsp, err := c.AddStubWithBody(ctx, contentType, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseAddStubResponse(rsp)
}

func (c *ClientWithResponses) AddStubWithResponse(ctx context.Context, body AddStubJSONRequestBody, reqEditors ...RequestEditorFn) (*AddStubResponse, error) {
	rsp, err := c.AddStub(ctx, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseAddStubResponse(rsp)
}

// BatchStubsDeleteWithBodyWithResponse request with arbitrary body returning *BatchStubsDeleteResponse
func (c *ClientWithResponses) BatchStubsDeleteWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*BatchStubsDeleteResponse, error) {
	rsp, err := c.BatchStubsDeleteWithBody(ctx, contentType, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseBatchStubsDeleteResponse(rsp)
}

func (c *ClientWithResponses) BatchStubsDeleteWithResponse(ctx context.Context, body BatchStubsDeleteJSONRequestBody, reqEditors ...RequestEditorFn) (*BatchStubsDeleteResponse, error) {
	rsp, err := c.BatchStubsDelete(ctx, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseBatchStubsDeleteResponse(rsp)
}

// SearchStubsWithBodyWithResponse request with arbitrary body returning *SearchStubsResponse
func (c *ClientWithResponses) SearchStubsWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*SearchStubsResponse, error) {
	rsp, err := c.SearchStubsWithBody(ctx, contentType, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseSearchStubsResponse(rsp)
}

func (c *ClientWithResponses) SearchStubsWithResponse(ctx context.Context, body SearchStubsJSONRequestBody, reqEditors ...RequestEditorFn) (*SearchStubsResponse, error) {
	rsp, err := c.SearchStubs(ctx, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseSearchStubsResponse(rsp)
}

// ListUnusedStubsWithResponse request returning *ListUnusedStubsResponse
func (c *ClientWithResponses) ListUnusedStubsWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*ListUnusedStubsResponse, error) {
	rsp, err := c.ListUnusedStubs(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseListUnusedStubsResponse(rsp)
}

// DeleteStubByIDWithResponse request returning *DeleteStubByIDResponse
func (c *ClientWithResponses) DeleteStubByIDWithResponse(ctx context.Context, uuid ID, reqEditors ...RequestEditorFn) (*DeleteStubByIDResponse, error) {
	rsp, err := c.DeleteStubByID(ctx, uuid, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseDeleteStubByIDResponse(rsp)
}

// ParseLivenessResponse parses an HTTP response from a LivenessWithResponse call
func ParseLivenessResponse(rsp *http.Response) (*LivenessResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &LivenessResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest MessageOK
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	}

	return response, nil
}

// ParseReadinessResponse parses an HTTP response from a ReadinessWithResponse call
func ParseReadinessResponse(rsp *http.Response) (*ReadinessResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &ReadinessResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest MessageOK
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	}

	return response, nil
}

// ParsePurgeStubsResponse parses an HTTP response from a PurgeStubsWithResponse call
func ParsePurgeStubsResponse(rsp *http.Response) (*PurgeStubsResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &PurgeStubsResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	return response, nil
}

// ParseListStubsResponse parses an HTTP response from a ListStubsWithResponse call
func ParseListStubsResponse(rsp *http.Response) (*ListStubsResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &ListStubsResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest StubList
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	}

	return response, nil
}

// ParseAddStubResponse parses an HTTP response from a AddStubWithResponse call
func ParseAddStubResponse(rsp *http.Response) (*AddStubResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &AddStubResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest struct {
			union json.RawMessage
		}
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	}

	return response, nil
}

// ParseBatchStubsDeleteResponse parses an HTTP response from a BatchStubsDeleteWithResponse call
func ParseBatchStubsDeleteResponse(rsp *http.Response) (*BatchStubsDeleteResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &BatchStubsDeleteResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	return response, nil
}

// ParseSearchStubsResponse parses an HTTP response from a SearchStubsWithResponse call
func ParseSearchStubsResponse(rsp *http.Response) (*SearchStubsResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &SearchStubsResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest SearchResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	}

	return response, nil
}

// ParseListUnusedStubsResponse parses an HTTP response from a ListUnusedStubsWithResponse call
func ParseListUnusedStubsResponse(rsp *http.Response) (*ListUnusedStubsResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &ListUnusedStubsResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest StubList
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	}

	return response, nil
}

// ParseDeleteStubByIDResponse parses an HTTP response from a DeleteStubByIDWithResponse call
func ParseDeleteStubByIDResponse(rsp *http.Response) (*DeleteStubByIDResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &DeleteStubByIDResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	return response, nil
}
