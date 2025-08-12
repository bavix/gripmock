package app

import (
	"github.com/gripmock/stuber"

	errorFormatter "github.com/bavix/gripmock/v3/internal/infra/errors"
)

func stubNotFoundError(expect stuber.Query, result *stuber.Result) error {
	formatter := errorFormatter.NewStubNotFoundFormatter()

	return formatter.FormatV1(expect, result)
}
