package remoteapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

//nolint:containedctx
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Session    string
	Context    context.Context
}

type HistoryCall struct {
	Service         string
	Method          string
	Session         string
	Requests        []map[string]any
	Responses       []any
	ResponseHeaders map[string]string
	Error           string
	Code            uint32
	ElapsedMS       int64
	StubID          uuid.UUID
	Timestamp       time.Time
}

const maxErrorBodyBytes = 4096

type HistoryFilter struct {
	Service   string
	Method    string
	Limit     int
	Offset    int
	ErrorOnly bool
}

func (f HistoryFilter) query() url.Values {
	q := url.Values{}

	if f.Service != "" {
		q.Set("service", f.Service)
	}

	if f.Method != "" {
		q.Set("method", f.Method)
	}

	if f.Limit > 0 {
		q.Set("limit", strconv.Itoa(f.Limit))
	}

	if f.Offset > 0 {
		q.Set("offset", strconv.Itoa(f.Offset))
	}

	if f.ErrorOnly {
		q.Set("error", "true")
	}

	return q
}

//nolint:funcorder
func (c Client) getHTTPClient() *http.Client {
	cli := c.HTTPClient
	if cli == nil {
		cli = http.DefaultClient
	}

	transport := cli.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	return &http.Client{
		Transport:     &gzipRoundTripper{next: transport},
		CheckRedirect: cli.CheckRedirect,
		Jar:           cli.Jar,
		Timeout:       cli.Timeout,
	}
}

type gzipRoundTripper struct {
	next http.RoundTripper
}

func (rt *gzipRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil && req.Body != http.NoBody {
		origBody, err := io.ReadAll(req.Body)
		_ = req.Body.Close()

		if err != nil {
			return nil, err
		}

		var buf bytes.Buffer

		gw := gzip.NewWriter(&buf)
		if _, err := gw.Write(origBody); err != nil {
			return nil, err
		}

		if err := gw.Close(); err != nil {
			return nil, err
		}

		req.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
		req.ContentLength = int64(buf.Len())
		req.Header.Set("Content-Encoding", "gzip")
	}

	resp, err := rt.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			_ = resp.Body.Close()

			return nil, err
		}

		resp.Body = reader
	}

	return resp, nil
}

//nolint:funcorder
func (c Client) getContext() context.Context {
	if c.Context != nil {
		return c.Context
	}

	return context.Background()
}

//nolint:funcorder
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

//nolint:funcorder
func (c Client) buildAPIURL(path string) (string, error) {
	apiURL, err := url.JoinPath(c.BaseURL, path)
	if err != nil {
		return "", fmt.Errorf("sdk: failed to build request URL: %w", err)
	}

	return apiURL, nil
}

//nolint:funcorder // GET requests go through sendRequestQuery instead.
func (c Client) sendRequest(method, path string, body []byte, contentType string) (*http.Response, error) {
	return c.sendRequestQuery(method, path, nil, body, contentType)
}

//nolint:funcorder
func (c Client) sendRequestQuery(
	method, path string,
	query url.Values,
	body []byte,
	contentType string,
) (*http.Response, error) {
	apiURL, err := c.buildAPIURL(path)
	if err != nil {
		return nil, err
	}

	if len(query) > 0 {
		apiURL += "?" + query.Encode()
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return describeFailure("add stubs", resp)
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return nil
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("sdk: batch delete stubs failed with status %d", resp.StatusCode) //nolint:err113
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return describeFailure("upload descriptors", resp)
	}

	return nil
}

func (c Client) FetchHistory() ([]HistoryCall, error) {
	calls, _, err := c.FetchHistoryFiltered(HistoryFilter{})

	return calls, err
}

func (c Client) FetchHistoryFiltered(filter HistoryFilter) ([]HistoryCall, int, error) {
	resp, err := c.sendRequestQuery(http.MethodGet, "api/history", filter.query(), nil, "")
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, describeFailure("fetch history", resp)
	}

	out, err := decodeHistory(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	return out, totalFromHeader(resp.Header, len(out)), nil
}

func (c Client) PurgeHistory() error {
	resp, err := c.sendRequest(http.MethodDelete, "api/history", nil, "")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return describeFailure("purge history", resp)
	}

	return nil
}

func decodeHistory(body io.Reader) ([]HistoryCall, error) {
	var list []struct {
		Service         *string             `json:"service"`
		Method          *string             `json:"method"`
		Session         *string             `json:"session"`
		Requests        *[]map[string]any   `json:"requests"`
		Responses       *[]any              `json:"responses"`
		ResponseHeaders *map[string]string  `json:"responseHeaders"`
		Code            *uint32             `json:"code"`
		Error           *string             `json:"error"`
		ElapsedMS       *int64              `json:"elapsedMs"`
		StubID          *openapi_types.UUID `json:"stubId"`
		Timestamp       *time.Time          `json:"timestamp"`
	}
	if err := json.NewDecoder(body).Decode(&list); err != nil {
		return nil, fmt.Errorf("sdk: failed to decode history: %w", err)
	}

	out := make([]HistoryCall, len(list))
	for i, call := range list {
		out[i] = HistoryCall{
			Service:         ptrOrZero(call.Service),
			Method:          ptrOrZero(call.Method),
			Session:         ptrOrZero(call.Session),
			Requests:        ptrOrZero(call.Requests),
			Responses:       ptrOrZero(call.Responses),
			ResponseHeaders: ptrOrZero(call.ResponseHeaders),
			Code:            ptrOrZero(call.Code),
			Error:           ptrOrZero(call.Error),
			ElapsedMS:       ptrOrZero(call.ElapsedMS),
			StubID:          ptrOrZero(call.StubID),
			Timestamp:       ptrOrZero(call.Timestamp),
		}
	}

	return out, nil
}

func totalFromHeader(h http.Header, fallback int) int {
	raw := h.Get("X-Total-Count")
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 {
		return fallback
	}

	return parsed
}

func ptrOrZero[T any](p *T) T { //nolint:ireturn
	if p == nil {
		var zero T

		return zero
	}

	return *p
}

func describeFailure(op string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))

	detail := strings.TrimSpace(string(body))
	if detail == "" {
		return fmt.Errorf("sdk: %s failed with status %d", op, resp.StatusCode) //nolint:err113
	}

	return fmt.Errorf("sdk: %s failed with status %d: %s", op, resp.StatusCode, detail) //nolint:err113
}
