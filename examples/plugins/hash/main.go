package main

import (
	"crypto/md5" //nolint:gosec // md5 kept for deprecated example
	"crypto/sha256"
	"encoding/hex"
	"hash/crc32"

	"github.com/bavix/gripmock/v3/pkg/plugins"
)

func Register(reg plugins.Registry) {
	reg.AddPlugin(plugins.PluginInfo{
		Name:         "hash",
		Source:       "examples/plugins/hash",
		Version:      "v1.2.3",
		Kind:         "external",
		Capabilities: []string{"template-funcs"},
		Authors: []plugins.Author{
			{Name: "Alice Smith", Contact: "alice@example.com"},
		},
		Description: "hash helpers",
	}, []plugins.SpecProvider{
		plugins.Specs(
			plugins.FuncSpec{
				Name: "crc32",
				Fn: func(s string) uint32 {
					return crc32.ChecksumIEEE([]byte(s))
				},
				Description: "crc32 checksum",
				Group:       "crc",
			},
			plugins.FuncSpec{
				Name: "sha256",
				Fn: func(s string) string {
					sum := sha256.Sum256([]byte(s))

					return hex.EncodeToString(sum[:])
				},
				Description: "sha256 hex checksum",
				Group:       "sha",
			},
			plugins.FuncSpec{
				Name: "md5",
				Fn: func(s string) string {
					sum := md5.Sum([]byte(s)) //nolint:gosec // md5 is intentionally shown as deprecated example

					return hex.EncodeToString(sum[:])
				},
				Description: "md5 hex checksum",
				Group:       "md5",
				Replacement: "sha256",
			},
		),
	})
}
