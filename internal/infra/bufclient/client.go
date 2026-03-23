package bufclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/cockroachdb/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/config"
)

const (
	bsrModuleParts = 3
	defaultRef     = "main"
	fdsEndpoint    = "/buf.registry.module.v1.FileDescriptorSetService/GetFileDescriptorSet"
)

type Client interface {
	FetchDescriptorSet(ctx context.Context, module, version string) (*descriptorpb.FileDescriptorSet, error)
}

type client struct {
	config config.Config

	httpClient *http.Client
}

type moduleRef struct {
	remote string
	owner  string
	repo   string
}

type getFDSRequest struct {
	ResourceRef                   resourceRef `json:"resourceRef"`
	ExcludeImports                bool        `json:"excludeImports"`
	IncludeSourceCodeInfo         bool        `json:"includeSourceCodeInfo"`
	IncludeSourceRetentionOptions bool        `json:"includeSourceRetentionOptions"`
}

type resourceRef struct {
	Name resourceName `json:"name"`
}

type resourceName struct {
	Owner  string `json:"owner"`
	Module string `json:"module"`
	Ref    string `json:"ref"`
}

type getFDSResponse struct {
	FileDescriptorSet json.RawMessage `json:"fileDescriptorSet"`
}

//nolint:ireturn
func NewClient(cfg config.Config) Client {
	httpClient := &http.Client{Timeout: cfg.BSRTimeout}
	if cfg.BSRToken != "" {
		httpClient.Transport = &authInterceptor{
			transport: http.DefaultTransport,
			token:     cfg.BSRToken,
		}
	}

	return &client{config: cfg, httpClient: httpClient}
}

func (c *client) FetchDescriptorSet(ctx context.Context, module, version string) (*descriptorpb.FileDescriptorSet, error) {
	mod, err := parseModule(module)
	if err != nil {
		return nil, err
	}

	baseURL := c.config.BSRBaseURL
	if baseURL == "" {
		baseURL = "https://" + mod.remote
	}

	ref := version
	if ref == "" {
		ref = defaultRef
	}

	requestBody, err := marshalRequest(mod, ref)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(baseURL, "/") + fdsEndpoint

	body, err := c.doRequest(ctx, endpoint, requestBody)
	if err != nil {
		return nil, err
	}

	return unmarshalResponse(body)
}

func marshalRequest(mod *moduleRef, ref string) ([]byte, error) {
	request := getFDSRequest{
		ResourceRef: resourceRef{
			Name: resourceName{Owner: mod.owner, Module: mod.repo, Ref: ref},
		},
		ExcludeImports:                false,
		IncludeSourceCodeInfo:         false,
		IncludeSourceRetentionOptions: true,
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal BSR request")
	}

	return payload, nil
}

func (c *client) doRequest(ctx context.Context, endpoint string, requestBody []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, errors.Wrap(err, "failed to build BSR request")
	}

	req.Header.Set("Connect-Protocol-Version", "1")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute BSR request")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read BSR response")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("BSR request failed (%s): %s", resp.Status, string(body))
	}

	return body, nil
}

func unmarshalResponse(body []byte) (*descriptorpb.FileDescriptorSet, error) {
	response := getFDSResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, errors.Wrap(err, "failed to decode BSR response")
	}

	if len(response.FileDescriptorSet) == 0 {
		return nil, errors.New("BSR returned empty descriptor set")
	}

	fds := &descriptorpb.FileDescriptorSet{}
	if err := protojson.Unmarshal(response.FileDescriptorSet, fds); err != nil {
		return nil, errors.Wrap(err, "failed to parse descriptor set from BSR response")
	}

	return fds, nil
}

func parseModule(module string) (*moduleRef, error) {
	parts := strings.SplitN(module, "/", bsrModuleParts)
	if len(parts) < bsrModuleParts {
		return nil, errors.Errorf("invalid BSR module: %s", module)
	}

	return &moduleRef{remote: parts[0], owner: parts[1], repo: parts[2]}, nil
}

type authInterceptor struct {
	transport http.RoundTripper
	token     string
}

func (a *authInterceptor) RoundTrip(req *http.Request) (*http.Response, error) {
	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}

	return a.transport.RoundTrip(req)
}
