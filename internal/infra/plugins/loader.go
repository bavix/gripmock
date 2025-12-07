package plugins

import (
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/rs/zerolog"

	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

type Loader struct {
	paths []string
}

func NewLoader(paths []string) *Loader {
	return &Loader{paths: paths}
}

func (l *Loader) WithGlob(glob string) *Loader {
	if glob == "" {
		return l
	}

	matches, err := filepath.Glob(glob)
	if err != nil {
		return l
	}

	l.paths = append(l.paths, matches...)

	return l
}

func (l *Loader) Load(reg pkgplugins.Registry) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Str("component", "plugin-loader").Logger()

	for _, p := range l.expandPaths() {
		stat, err := os.Stat(p)
		if err != nil {
			logger.Warn().Str("path", p).Err(err).Msg("plugin load skip")

			continue
		}

		if stat.IsDir() {
			continue
		}

		lp, err := plugin.Open(p)
		if err != nil {
			logger.Warn().Str("path", p).Err(err).Msg("plugin load skip")

			continue
		}

		sym, err := lp.Lookup("Register")
		if err != nil {
			logger.Warn().Str("path", p).Err(err).Msg("plugin register symbol not found")

			continue
		}

		info := pkgplugins.PluginInfo{
			Name:         strings.TrimSuffix(filepath.Base(p), filepath.Ext(p)),
			Source:       p,
			Kind:         "external",
			Capabilities: []string{"template-funcs"},
		}

		if fn, ok := sym.(func(pkgplugins.Registry) error); ok {
			if err := fn(reg); err == nil {
				continue
			}
		}

		if fn, ok := sym.(func(pkgplugins.Registry)); ok {
			fn(reg)
		} else {
			logger.Warn().Str("path", p).Msg("plugin register symbol has unsupported signature")
		}

		if !existsPlugin(reg, info.Name) {
			reg.AddPlugin(info, nil)
		}
	}
}

func (l *Loader) expandPaths() []string {
	paths := make([]string, 0, len(l.paths))
	for _, p := range l.paths {
		if stat, err := os.Stat(p); err == nil && stat.IsDir() {
			matches, globErr := filepath.Glob(filepath.Join(p, "*.so"))
			if globErr == nil {
				paths = append(paths, matches...)
			}

			continue
		}

		if strings.Contains(p, "*") {
			matches, err := filepath.Glob(p)
			if err == nil {
				paths = append(paths, matches...)
			}

			continue
		}

		paths = append(paths, p)
	}

	return paths
}

func existsPlugin(reg pkgplugins.Registry, name string) bool {
	for _, info := range reg.Plugins() {
		if info.Name == name {
			return true
		}
	}

	return false
}
