package deps

import (
	"io/fs"

	"github.com/cockroachdb/errors"

	gripmockui "github.com/bavix/gripmock-ui"
)

func (b *Builder) ui() (fs.FS, error) {
	assets, err := gripmockui.Assets()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get UI assets")
	}

	return assets, nil
}
