//go:build windows

package main

import "github.com/bavix/gripmock/v3/pkg/plugins/cshared/export"

//nolint:gochecknoinits
func init() { export.Use(Register) }

func main() {}
