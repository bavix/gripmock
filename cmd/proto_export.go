package cmd

import (
	"slices"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/infra/protobundle"
)

func registerProtoExport(parent *cobra.Command) {
	var (
		roots       []string
		importRoots []string
		output      string
		include     []string
		exclude     []string
	)

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Compile proto files into a descriptor bundle (.pb / .pbs)",
		Long: `Compile .proto files from one or more root directories into a single
FileDescriptorSet file using protocompile. No system protoc required.

Output format is determined by the --out file extension:
  .pb  — raw protobuf (FileDescriptorSet)
  .pbs — S2-compressed protobuf (block mode, best compression)

When the same relative path exists in multiple roots, the file with
the most modern syntax wins (edition "2024" > edition "2023" > proto3 > proto2).
Among equal syntax, the first root wins.

The --import-root flag adds import paths for compilation only: files from
these roots are NOT discovered or included in the output, but can be
resolved as transitive dependencies during compilation.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runProtoExport(cmd, roots, importRoots, output, include, exclude)
		},
	}

	exportCmd.Flags().StringSliceVar(&roots, "root", nil, "Discovery and import root directories (required, repeatable)")
	exportCmd.Flags().StringSliceVar(&importRoots, "import-root", nil, "Import-only root directories (not discovered, not included in output)")
	exportCmd.Flags().StringVar(&output, "out", "", "Output file path: .pb (raw) or .pbs (S2-compressed) (required)")
	exportCmd.Flags().StringSliceVar(&include, "include", nil, "Glob patterns to include (default: **/*.proto)")
	exportCmd.Flags().StringSliceVar(&exclude, "exclude", nil, "Glob patterns to exclude")

	_ = exportCmd.MarkFlagRequired("root")
	_ = exportCmd.MarkFlagRequired("out")

	parent.AddCommand(exportCmd)
}

func runProtoExport(cmd *cobra.Command, roots, importRoots []string, output string, include, exclude []string) error {
	ctx := cmd.Context()
	logger := zerolog.Ctx(ctx)

	logger.Info().Strs("roots", roots).Strs("import-roots", importRoots).Str("out", output).Msg("discovering proto files")

	result, err := protobundle.Discover(protobundle.DiscoverParams{
		Roots:   roots,
		Include: include,
		Exclude: exclude,
	})
	if err != nil {
		return errors.Wrap(err, "discovery failed")
	}

	files := result.Sorted()

	logger.Info().
		Int("files", len(files)).
		Int("skipped", len(result.Skipped)).
		Int("unsupported_edition", len(result.UnsupportedEdition)).
		Msg("discovery complete")

	for _, s := range result.Skipped {
		logger.Debug().Str("skipped", s).Msg("file shadowed by deduplication")
	}

	for _, s := range result.UnsupportedEdition {
		logger.Debug().Str("file", s).Msg("skipped: unsupported edition")
	}

	if len(files) == 0 {
		return errors.New("no proto files found")
	}

	compileRoots := append(slices.Clone(roots), importRoots...)

	logger.Info().Int("files", len(files)).Msg("compiling proto files")

	fds, err := protobundle.Compile(ctx, protobundle.CompileParams{
		Roots: compileRoots,
		Files: files,
	})
	if err != nil {
		return errors.Wrap(err, "compilation failed")
	}

	fds = filterToDiscovered(fds, result.Files)

	logger.Info().
		Int("descriptors", len(fds.GetFile())).
		Str("out", output).
		Msg("writing descriptor set")

	if err = protobundle.Write(fds, output); err != nil {
		return errors.Wrap(err, "write failed")
	}

	logger.Info().Str("out", output).Msg("done")

	return nil
}

func filterToDiscovered(fds *descriptorpb.FileDescriptorSet, discovered map[string]string) *descriptorpb.FileDescriptorSet {
	filtered := &descriptorpb.FileDescriptorSet{
		File: make([]*descriptorpb.FileDescriptorProto, 0, len(fds.GetFile())),
	}

	for _, f := range fds.GetFile() {
		if _, ok := discovered[f.GetName()]; ok {
			filtered.File = append(filtered.File, f)
		}
	}

	return filtered
}
