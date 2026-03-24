package bufclient

import (
	"context"
	"net/url"
	"strings"

	"github.com/cockroachdb/errors"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/config"
)

const defaultBufBaseURL = "https://buf.build"

type Router struct {
	bufClient  *Client
	selfClient *Client
}

func NewRouter(cfg config.BSRConfig) *Router {
	if cfg.Buf.BaseURL == nil {
		cfg.Buf.BaseURL, _ = url.Parse(defaultBufBaseURL)
	}

	return &Router{
		bufClient:  NewClient(cfg.Buf),
		selfClient: NewClient(cfg.Self),
	}
}

func (r *Router) FetchDescriptorSet(ctx context.Context, module, version string) (*descriptorpb.FileDescriptorSet, error) {
	remote, owner, repo, err := parseModule(module)
	if err != nil {
		return nil, err
	}

	client := r.selectClient(remote)

	return client.FetchDescriptorSet(ctx, owner, repo, version)
}

func (r *Router) selectClient(remote string) *Client {
	if r.selfClient.BaseURL == nil || r.selfClient.BaseURL.Host == "" {
		return r.bufClient
	}

	if strings.EqualFold(r.selfClient.BaseURL.Host, remote) {
		return r.selfClient
	}

	return r.bufClient
}

func parseModule(module string) (string, string, string, error) {
	raw := strings.TrimSpace(module)

	// Ensure url.Parse puts host/owner/repo into Host+Path, not just Path.
	if !strings.Contains(raw, "://") {
		raw = "https://" + strings.TrimPrefix(raw, "//")
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "", "", "", errors.Errorf("invalid BSR module: %s", module)
	}

	parts := strings.SplitN(strings.Trim(parsed.Path, "/"), "/", 3) //nolint:mnd
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", errors.Errorf("invalid BSR module: %s", module)
	}

	return parsed.Host, parts[0], parts[1], nil
}
