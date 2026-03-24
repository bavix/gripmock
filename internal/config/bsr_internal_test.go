package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadBSRConfigBufPrefix(t *testing.T) {
	t.Setenv("BSR_BUF_BASE_URL", "https://buf.build")
	t.Setenv("BSR_BUF_TOKEN", "token-default")
	t.Setenv("BSR_BUF_TIMEOUT", "7s")

	cfg := Load()
	profile := cfg.BSR.Buf

	require.Equal(t, "https://buf.build", profile.BaseURL.String())
	require.Equal(t, "token-default", profile.Token)
	require.Equal(t, 7*time.Second, profile.Timeout)
}

func TestLoadBSRConfigSelfPrefix(t *testing.T) {
	t.Setenv("BSR_SELF_BASE_URL", "https://bsr.company.local")
	t.Setenv("BSR_SELF_TOKEN", "token-pay")
	t.Setenv("BSR_SELF_TIMEOUT", "9s")

	cfg := Load()
	profile := cfg.BSR.Self

	require.Equal(t, "https://bsr.company.local", profile.BaseURL.String())
	require.Equal(t, "token-pay", profile.Token)
	require.Equal(t, 9*time.Second, profile.Timeout)
}

func TestLoadBSRConfigBufFallback(t *testing.T) {
	t.Setenv("BSR_BUF_TIMEOUT", "6s")

	cfg := Load()
	profile := cfg.BSR.Buf

	require.Nil(t, profile.BaseURL)
	require.Empty(t, profile.Token)
	require.Equal(t, 6*time.Second, profile.Timeout)
}
