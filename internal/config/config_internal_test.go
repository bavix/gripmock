package config

import (
	"testing"
)

func TestParseHistoryDefaults(t *testing.T) {
	// clear env via Setenv to ensure defaults
	t.Setenv("HISTORY_LIMIT", "")
	t.Setenv("HISTORY_REDACT_KEYS", "")
	t.Setenv("HISTORY_MESSAGE_MAX_BYTES", "")
	t.Setenv("HISTORY_ENABLED", "")

	cfg := Load()
	if !cfg.HistoryEnabled {
		t.Fatalf("history should be enabled by default")
	}

	if cfg.HistoryLimit.Int64() != 64*1024*1024 {
		t.Fatalf("unexpected default limit: %d", cfg.HistoryLimit.Int64())
	}

	if cfg.HistoryMessageMaxBytes != 262144 {
		t.Fatalf("unexpected default max bytes: %d", cfg.HistoryMessageMaxBytes)
	}

	if len(cfg.HistoryRedactKeys) != 0 {
		t.Fatalf("expected no redact keys by default")
	}
}

func TestParseHistoryEnv(t *testing.T) {
	toGiB := int64(1024 * 1024 * 1024)

	t.Setenv("HISTORY_LIMIT", "1G")
	t.Setenv("HISTORY_REDACT_KEYS", "password,token,secret")
	t.Setenv("HISTORY_MESSAGE_MAX_BYTES", "1024")
	t.Setenv("HISTORY_ENABLED", "false")

	cfg := Load()
	if cfg.HistoryEnabled {
		t.Fatalf("history should be disabled by env")
	}

	if cfg.HistoryLimit.Int64() != toGiB {
		t.Fatalf("unexpected limit: %d", cfg.HistoryLimit.Int64())
	}

	if cfg.HistoryMessageMaxBytes != 1024 {
		t.Fatalf("unexpected max bytes: %d", cfg.HistoryMessageMaxBytes)
	}

	keys := cfg.HistoryRedactKeys
	if len(keys) != 3 || keys[0] != "password" || keys[1] != "token" || keys[2] != "secret" {
		t.Fatalf("unexpected redact keys: %#v", keys)
	}
}
