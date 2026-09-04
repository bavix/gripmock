package plugins

import (
	"context"
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

func (l *Loader) Load(ctx context.Context, reg pkgplugins.Registry) {
	logger := zerolog.Ctx(ctx)

	for _, p := range l.expandPaths() {
		if !l.loadOne(ctx, reg, logger, p) {
			return
		}
	}
}

func (l *Loader) loadOne(
	ctx context.Context,
	reg pkgplugins.Registry,
	logger *zerolog.Logger,
	path string,
) bool {
	stat, err := os.Stat(path)
	if err != nil {
		logger.Warn().Str("path", path).Err(err).Msg("plugin load skip")

		return true
	}

	if stat.IsDir() {
		return true
	}

	lp, supported := openPlugin(logger, path)
	if !supported {
		return false
	}

	if lp == nil {
		return true
	}

	sym, err := lp.Lookup("Register")
	if err != nil {
		logger.Warn().Str("path", path).Err(err).Msg("plugin register symbol not found")

		return true
	}

	registerPlugin(ctx, reg, logger, path, sym)

	return true
}

func registerPlugin(
	ctx context.Context,
	reg pkgplugins.Registry,
	logger *zerolog.Logger,
	path string,
	sym any,
) {
	if fn, ok := sym.(func(pkgplugins.Registry) error); ok {
		registerErr := fn(reg)
		if registerErr != nil {
			logger.Warn().Str("path", path).Err(registerErr).Msg("plugin register error, skipping")
		}

		return
	}

	if fn, ok := sym.(func(pkgplugins.Registry)); ok {
		fn(reg)

		return
	}

	logger.Warn().Str("path", path).Msg("plugin register symbol has unsupported signature")

	info := pkgplugins.PluginInfo{
		Name:         strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		Source:       path,
		Kind:         "external",
		Capabilities: []string{"template-funcs"},
	}

	if !existsPlugin(ctx, reg, info.Name) {
		reg.AddPlugin(info, nil)
	}
}

func openPlugin(logger *zerolog.Logger, path string) (*plugin.Plugin, bool) {
	lp, err := plugin.Open(path)
	if err == nil {
		return lp, true
	}

	if isPluginUnsupported(err) {
		logger.Error().
			Str("path", path).
			Err(err).
			Msg("plugin support is missing from this build (compiled without cgo); " +
				"use the docker image or build from source with CGO_ENABLED=1")

		return nil, false
	}

	logger.Warn().Str("path", path).Err(err).Msg("plugin load skip")

	return nil, true
}

func isPluginUnsupported(err error) bool {
	return err != nil && strings.Contains(err.Error(), "plugin: not implemented")
}

func (l *Loader) expandPaths() []string {
	paths := make([]string, 0, len(l.paths))
	for _, p := range l.paths {
		stat, err := os.Stat(p)
		if err == nil && stat.IsDir() {
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

func existsPlugin(ctx context.Context, reg pkgplugins.Registry, name string) bool {
	for _, info := range reg.Plugins(ctx) {
		if info.Name == name {
			return true
		}
	}

	return false
}
