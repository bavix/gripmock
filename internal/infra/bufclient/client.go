package bufclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/config"
)

const (
	fdsEndpoint    = "/buf.registry.module.v1.FileDescriptorSetService/GetFileDescriptorSet"
	defaultTimeout = 5 * time.Second
)

type Client struct {
	BaseURL *url.URL
	Token   string
	Timeout time.Duration
}

func NewClient(profile config.BSRProfile) *Client {
	return &Client{
		BaseURL: profile.BaseURL,
		Token:   profile.Token,
		Timeout: profile.Timeout,
	}
}

func (c *Client) FetchDescriptorSet(ctx context.Context, owner, module, ref string) (*descriptorpb.FileDescriptorSet, error) {
	if ref == "" {
		ref = "main"
	}

	payload, err := c.buildRequestPayload(owner, module, ref)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build request payload")
	}

	req, err := c.buildRequest(ctx, payload)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	return c.parseResponse(body)
}

func (c *Client) buildRequestPayload(owner, module, ref string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"resourceRef": map[string]any{
			"name": map[string]any{
				"owner":  owner,
				"module": module,
				"ref":    ref,
			},
		},
		"includeSourceRetentionOptions": true,
	})
}

func (c *Client) buildRequest(ctx context.Context, payload []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL.String()+fdsEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, errors.Wrap(err, "failed to build BSR request")
	}

	req.Header.Set("Connect-Protocol-Version", "1")
	req.Header.Set("Content-Type", "application/json")

	if token := strings.TrimSpace(c.Token); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return req, nil
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	//nolint:gosec
	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute BSR request")
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read BSR response")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("BSR request failed (%s): %s", resp.Status, string(body))
	}

	return body, nil
}

func (c *Client) parseResponse(body []byte) (*descriptorpb.FileDescriptorSet, error) {
	var envelope struct {
		FileDescriptorSet json.RawMessage `json:"fileDescriptorSet"`
	}

	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, errors.Wrap(err, "failed to decode BSR response")
	}

	if len(envelope.FileDescriptorSet) == 0 {
		return nil, errors.New("BSR returned empty descriptor set")
	}

	fds := &descriptorpb.FileDescriptorSet{}
	if err := protojson.Unmarshal(envelope.FileDescriptorSet, fds); err != nil {
		return nil, errors.Wrap(err, "failed to parse descriptor set from BSR response")
	}

	return fds, nil
}
