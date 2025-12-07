package stuber

import (
	templatepkg "github.com/bavix/gripmock/v3/internal/infra/template"
	"github.com/bavix/gripmock/v3/pkg/plugins"
)

func TemplateFunctions(reg plugins.Registry) map[string]any {
	return templatepkg.Functions(reg)
}
