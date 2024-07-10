package deps

import (
	"io/fs"

	gripmockui "github.com/bavix/gripmock-ui"
)

func (b *Builder) ui() (fs.FS, error) {
	return gripmockui.Assets()
}
