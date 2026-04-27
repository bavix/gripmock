package remoteapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Session    string
	Context    context.Context
}

type HistoryCall struct {
	Service   string
	Method    string
	Request   map[string]any
	Requests  []map[string]any
	Response  map[string]any
	Responses []map[string]any
	Error     string
	Code      uint32
	StubID    uuid.UUID
	Timestamp time.Time
}

type VerifyBadRequestError struct {
	Message string
}

func (e VerifyBadRequestError) Error() string {
	if e.Message == "" {
		return "verification failed"
	}

	return e.Message
}

func (c Client) getHTTPClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}

	return http.DefaultClient
}

func (c Client) getContext() context.Context {
	if c.Context != nil {
		return c.Context
	}

	return context.Background()
}

func (c Client) newRequest(method, requestURL string, body []byte, contentType string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(c.getContext(), method, requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if c.Session != "" {
		req.Header.Set("X-Gripmock-Session", c.Session)
	}

	return req, nil
}

func (c Client) buildAPIURL(path string) (string, error) {
	apiURL, err := url.JoinPath(c.BaseURL, path)
	if err != nil {
		return "", fmt.Errorf("sdk: failed to build request URL: %w", err)
	}

	return apiURL, nil
}

func (c Client) sendRequest(method, path string, body []byte, contentType string) (*http.Response, error) {
	apiURL, err := c.buildAPIURL(path)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(method, apiURL, body, contentType)
	if err != nil {
		return nil, fmt.Errorf("sdk: failed to create request: %w", err)
	}

	resp, err := c.getHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("sdk: failed to execute request: %w", err)
	}

	return resp, nil
}

func (c Client) AddStub(stub *stuber.Stub) error {
	return c.AddStubs([]*stuber.Stub{stub})
}

func (c Client) AddStubs(stubs []*stuber.Stub) error {
	if len(stubs) == 0 {
		return nil
	}

	body, err := json.Marshal(stubs)
	if err != nil {
		return fmt.Errorf("sdk: failed to marshal stubs: %w", err)
	}

	resp, err := c.sendRequest(
		http.MethodPost,
		"api/stubs",
		body,
		"application/json",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("sdk: add stubs failed with status %d", resp.StatusCode)
	}

	return nil
}

func (c Client) BatchDelete(ids []uuid.UUID) error {
	body, err := json.Marshal(ids)
	if err != nil {
		return fmt.Errorf("sdk: failed to marshal stub IDs: %w", err)
	}

	resp, err := c.sendRequest(
		http.MethodPost,
		"api/stubs/batchDelete",
		body,
		"application/json",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("sdk: batch delete stubs failed with status %d", resp.StatusCode)
	}

	return nil
}

func (c Client) UploadDescriptors(files []*descriptorpb.FileDescriptorProto) error {
	if len(files) == 0 {
		return nil
	}

	body, err := proto.Marshal(&descriptorpb.FileDescriptorSet{File: files})
	if err != nil {
		return fmt.Errorf("sdk: failed to marshal descriptor set: %w", err)
	}

	resp, err := c.sendRequest(
		http.MethodPost,
		"api/descriptors",
		body,
		"application/octet-stream",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("sdk: upload descriptors failed with status %d", resp.StatusCode)
	}

	return nil
}

func (c Client) FetchHistory() ([]HistoryCall, error) {
	resp, err := c.sendRequest(
		http.MethodGet,
		"api/history",
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sdk: fetch history failed with status %d", resp.StatusCode)
	}

	var list []struct {
		Service   *string             `json:"service"`
		Method    *string             `json:"method"`
		Request   *map[string]any     `json:"request"`
		Requests  *[]map[string]any   `json:"requests"`
		Response  *map[string]any     `json:"response"`
		Responses *[]map[string]any   `json:"responses"`
		Code      *uint32             `json:"code"`
		Error     *string             `json:"error"`
		StubID    *openapi_types.UUID `json:"stubId"`
		Timestamp *time.Time          `json:"timestamp"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("sdk: failed to decode history: %w", err)
	}

	out := make([]HistoryCall, len(list))
	for i, call := range list {
		out[i] = HistoryCall{
			Service:   ptrOrZero(call.Service),
			Method:    ptrOrZero(call.Method),
			Request:   ptrOrZero(call.Request),
			Requests:  ptrOrZero(call.Requests),
			Response:  ptrOrZero(call.Response),
			Responses: ptrOrZero(call.Responses),
			Code:      ptrOrZero(call.Code),
			Error:     ptrOrZero(call.Error),
			StubID:    uuid.UUID(ptrOrZero(call.StubID)),
			Timestamp: ptrOrZero(call.Timestamp),
		}
	}

	return out, nil
}

func (c Client) VerifyMethodCalled(service, method string, expectedCount int) error {
	body, err := json.Marshal(map[string]any{
		"service":       service,
		"method":        method,
		"expectedCount": expectedCount,
	})
	if err != nil {
		return fmt.Errorf("sdk: failed to marshal verify request: %w", err)
	}

	resp, err := c.sendRequest(
		http.MethodPost,
		"api/verify",
		body,
		"application/json",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		var errBody struct {
			Message *string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)

		return VerifyBadRequestError{Message: ptrOrZero(errBody.Message)}
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("sdk: verify request failed with status %d", resp.StatusCode)
	}

	return nil
}

func ptrOrZero[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}

	return *p
}
