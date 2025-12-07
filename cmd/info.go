package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/bavix/gripmock/v3/internal/deps"
	"github.com/bavix/gripmock/v3/internal/infra/build"
	"github.com/bavix/gripmock/v3/pkg/plugins"
)

func init() { //nolint:gochecknoinits
	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "Show gripmock info and loaded plugins",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInfoCmd(cmd.Context())
		},
	}

	rootCmd.AddCommand(infoCmd)
}

type uiStats struct {
	totalPlugins int
	totalFuncs   int
	builtin      int
	external     int
}

type pluginGroup struct {
	info  plugins.PluginInfo
	funcs []plugins.FunctionInfo // all functions
}

type uiContext struct {
	width       int
	titleStyle  lipgloss.Style
	keyStyle    lipgloss.Style
	valStyle    lipgloss.Style
	dimStyle    lipgloss.Style
	tagBuiltin  lipgloss.Style
	tagExternal lipgloss.Style
}

const (
	labelBuiltin          = "gripmock"
	labelExternal         = "external"
	defaultWidth          = 100
	builtinFuncMultiplier = 8
	dash                  = "—"
)

func renderSummary(groups map[string]*pluginGroup) string {
	ctx := newUIContext()

	stats := countStats(groups)

	pad := func(label string) string {
		return fmt.Sprintf("%-9s:", label)
	}

	lines := []string{
		fmt.Sprintf("%s %s", ctx.titleStyle.Render(pad("GripMock")), ctx.valStyle.Render(build.Version)),
		fmt.Sprintf("%s %s", ctx.keyStyle.Render(pad("Go")), ctx.valStyle.Render(runtime.Version())),
		fmt.Sprintf("%s %s", ctx.keyStyle.Render(pad("Platform")), ctx.valStyle.Render(runtime.GOOS+"/"+runtime.GOARCH)),
		fmt.Sprintf("%s %s", ctx.keyStyle.Render(pad("Plugins")), ctx.valStyle.Render(strconv.Itoa(stats.totalPlugins))),
		fmt.Sprintf("%s %s", ctx.keyStyle.Render(pad("Functions")), ctx.valStyle.Render(strconv.Itoa(stats.totalFuncs))),
		fmt.Sprintf("%s %s", ctx.keyStyle.Render(pad("Builtin")), ctx.tagBuiltin.Render(strconv.Itoa(stats.builtin))),
		fmt.Sprintf("%s %s", ctx.keyStyle.Render(pad("External")), ctx.tagExternal.Render(strconv.Itoa(stats.external))),
	}

	return strings.Join(lines, "\n")
}

func groupPlugins(groupsData []plugins.PluginWithFuncs) ([]string, map[string]*pluginGroup) {
	order := make([]string, 0, len(groupsData))
	groups := make(map[string]*pluginGroup, len(groupsData))

	builtinList := make([]plugins.PluginWithFuncs, 0, len(groupsData))
	others := make([]plugins.PluginWithFuncs, 0, len(groupsData))

	sort.Slice(groupsData, func(i, j int) bool {
		return groupsData[i].Plugin.Name < groupsData[j].Plugin.Name
	})

	for _, g := range groupsData {
		if g.Plugin.Source == labelBuiltin {
			builtinList = append(builtinList, g)

			continue
		}

		others = append(others, g)
	}

	if agg, ok := aggregateBuiltins(builtinList); ok {
		order = append(order, agg.Plugin.Name)
		groups[agg.Plugin.Name] = &pluginGroup{info: agg.Plugin, funcs: agg.Funcs}
	}

	for _, g := range others {
		name := g.Plugin.Name

		order = append(order, name)
		groups[name] = &pluginGroup{
			info:  g.Plugin,
			funcs: g.Funcs,
		}
	}

	return order, groups
}

func aggregateBuiltins(list []plugins.PluginWithFuncs) (plugins.PluginWithFuncs, bool) {
	if len(list) == 0 {
		return plugins.PluginWithFuncs{}, false
	}

	funcs := make([]plugins.FunctionInfo, 0, len(list)*builtinFuncMultiplier)
	seen := make(map[string]struct{}, len(list)*builtinFuncMultiplier)

	for _, g := range list {
		for _, f := range g.Funcs {
			if _, ok := seen[f.Name]; ok {
				continue
			}

			seen[f.Name] = struct{}{}
			funcs = append(funcs, f)
		}
	}

	return plugins.PluginWithFuncs{Plugin: list[0].Plugin, Funcs: funcs}, true
}

func renderPluginGroup(g *pluginGroup) string {
	ctx := newUIContext()

	categoryTag := classifyPlugin(g.info, ctx)
	versionClean := displayOrDash(g.info.Version)

	var b strings.Builder

	fmt.Fprintf(&b, "%s %s %s\n",
		ctx.titleStyle.Render(g.info.Name),
		categoryTag,
		ctx.dimStyle.Render(versionClean),
	)

	renderKeyValue(&b, ctx, "Source", formatSource(g.info.Source))

	if len(g.info.Depends) > 0 {
		deps := append([]string(nil), g.info.Depends...)
		sort.Strings(deps)
		renderKeyValue(&b, ctx, "Depends", strings.Join(deps, ", "))
	}

	if authorsLine := formatAuthorsLine(ctx, g.info.Authors, nil, ""); authorsLine != "" {
		fmt.Fprintf(&b, "%s\n", authorsLine)
	}

	if g.info.Description != "" {
		renderKeyValue(&b, ctx, "Description", g.info.Description)
	}

	if caps := renderCapabilities(g.info.Capabilities); caps != "" {
		renderKeyValue(&b, ctx, "Capabilities", caps)
	}

	renderGroupedFunctions(&b, ctx, g.funcs)

	return b.String()
}

func renderKeyValue(b *strings.Builder, ctx uiContext, key, val string) {
	fmt.Fprintf(b, "%s%s %s\n",
		ctx.keyStyle.Render(key),
		ctx.dimStyle.Render(":"),
		ctx.valStyle.Render(val),
	)
}

func renderCapabilities(caps []string) string {
	if len(caps) == 0 {
		return ""
	}

	sorted := append([]string(nil), caps...)
	sort.Strings(sorted)

	return strings.Join(sorted, ", ")
}

func renderFunctionsLine(ctx uiContext, funcs []plugins.FunctionInfo) string {
	if len(funcs) == 0 {
		return ""
	}

	names := make([]string, 0, len(funcs))
	for _, f := range funcs {
		label := f.Name

		if f.Deactivated {
			label += " [deactivated]"
		}

		if repl := strings.TrimSpace(f.Replacement); repl != "" {
			label += fmt.Sprintf(" [deprecated → %s]", repl)
		}

		if f.Decorates != "" {
			label = fmt.Sprintf("%s (decorates: %s)", f.Name, formatDecorates(f))
		}

		names = append(names, label)
	}

	sort.Strings(names)

	return fmt.Sprintf("%s %s",
		ctx.keyStyle.Render(fmt.Sprintf("Functions (%d):", len(funcs))),
		ctx.valStyle.Render(strings.Join(names, ", ")),
	)
}

func renderGroupedFunctions(b *strings.Builder, ctx uiContext, funcs []plugins.FunctionInfo) {
	if len(funcs) == 0 {
		return
	}

	grouped := make(map[string][]string)
	plain := make([]plugins.FunctionInfo, 0, len(funcs))

	for _, f := range funcs {
		if strings.TrimSpace(f.Group) == "" {
			plain = append(plain, f)

			continue
		}

		grouped[f.Group] = append(grouped[f.Group], formatFunctionLabel(f))
	}

	if len(grouped) > 0 {
		groupNames := make([]string, 0, len(grouped))
		for g := range grouped {
			groupNames = append(groupNames, g)
		}

		sort.Strings(groupNames)

		for _, gname := range groupNames {
			names := grouped[gname]
			sort.Strings(names)
			prefix := ctx.dimStyle.Render(fmt.Sprintf("%s (%d)", gname, len(names)))
			fmt.Fprintf(b, "%s%s %s\n",
				prefix,
				ctx.dimStyle.Render(":"),
				ctx.valStyle.Render(strings.Join(names, ", ")),
			)
		}
	}

	if line := renderFunctionsLine(ctx, plain); line != "" {
		fmt.Fprintf(b, "%s\n", line)
	}
}

func formatFunctionLabel(f plugins.FunctionInfo) string {
	label := f.Name

	if f.Deactivated {
		label += " [deactivated]"
	}

	if repl := strings.TrimSpace(f.Replacement); repl != "" {
		label += fmt.Sprintf(" [deprecated → %s]", repl)
	}

	if f.Decorates != "" {
		label = fmt.Sprintf("%s (decorates: %s)", f.Name, formatDecorates(f))
	}

	return label
}

func runInfoCmd(ctx context.Context) error {
	builder := deps.NewBuilder(
		deps.WithDefaultConfig(),
		deps.WithPlugins(pluginsFlag),
	)

	ctx = builder.Logger(ctx)

	builder.LoadPlugins(ctx)

	raw := builder.PluginInfos()

	order, groups := groupPlugins(raw)

	fnCount := 0
	for _, g := range groups {
		fnCount += len(g.funcs)
	}

	var out strings.Builder
	out.WriteString(renderSummary(groups))
	out.WriteString("\n\n")

	for _, name := range order {
		out.WriteString(renderPluginGroup(groups[name]))
		out.WriteString("\n")
	}

	_, _ = os.Stdout.WriteString(out.String())

	return nil
}

func newUIContext() uiContext {
	width := termWidth()

	return uiContext{
		width:      width,
		titleStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
		keyStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		valStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true),
		dimStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		tagBuiltin: lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true),
		tagExternal: lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).Bold(true),
	}
}

func displayOrDash(v string) string {
	if strings.TrimSpace(v) == "" {
		return "—"
	}

	return v
}

func formatSource(src string) string {
	s := strings.TrimSpace(src)
	if s == "" {
		return "—"
	}

	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "file://") {
		return s
	}

	if strings.HasPrefix(s, "/") {
		return "file://" + s
	}

	if s == labelBuiltin {
		return labelBuiltin
	}

	return filepath.Clean("./" + s)
}

func formatDecorates(f plugins.FunctionInfo) string {
	name := strings.TrimSpace(f.Decorates)
	plug := strings.TrimSpace(f.DecoratesPlugin)

	if name == "" {
		return dash
	}

	if plug == "" {
		return name
	}

	return "@" + plug + "/" + name
}

func formatAuthorsLine(ctx uiContext, authors []plugins.Author, _ []string, _ string) string {
	items := make([]string, 0, len(authors))

	for _, a := range authors {
		name := strings.TrimSpace(a.Name)
		contact := strings.TrimSpace(a.Contact)

		if name == "" && contact == "" {
			continue
		}

		switch {
		case name != "" && contact != "":
			items = append(items, fmt.Sprintf("%s <%s>", name, contact))
		case name != "":
			items = append(items, name)
		default:
			items = append(items, contact)
		}
	}

	if len(items) == 0 {
		return ""
	}

	return fmt.Sprintf("%s %s", ctx.keyStyle.Render("Authors"), ctx.valStyle.Render(strings.Join(items, ", ")))
}

func classifyPlugin(info plugins.PluginInfo, ctx uiContext) string {
	name := strings.ToLower(info.Name)
	src := strings.ToLower(info.Source)

	switch {
	case name == labelBuiltin || src == labelBuiltin:
		return ctx.tagBuiltin.Render("builtin")
	case strings.HasSuffix(src, ".so") || strings.HasSuffix(name, ".so"):
		return ctx.tagExternal.Render(labelExternal)
	default:
		return ctx.tagExternal.Render(labelExternal)
	}
}

func countStats(groups map[string]*pluginGroup) uiStats {
	stats := uiStats{}
	for _, g := range groups {
		stats.totalPlugins++
		stats.totalFuncs += len(g.funcs)

		name := strings.ToLower(g.info.Name)
		src := strings.ToLower(g.info.Source)

		switch {
		case name == labelBuiltin || src == labelBuiltin:
			stats.builtin++
		case strings.HasSuffix(src, ".so") || strings.HasSuffix(name, ".so"):
			stats.external++
		default:
			stats.external++
		}
	}

	return stats
}

func termWidth() int {
	if cols, ok := os.LookupEnv("COLUMNS"); ok {
		if n, err := strconv.Atoi(cols); err == nil && n > 0 {
			return n
		}
	}

	return defaultWidth
}
