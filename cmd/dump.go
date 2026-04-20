package cmd

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/spf13/cobra"

	"github.com/bavix/gripmock/v3/internal/config"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

var ErrUnexpectedStatus = errors.New("unexpected status")

func init() { //nolint:gochecknoinits
	dumpCmd := &cobra.Command{
		Use:   "dump",
		Short: "Export stubs to files",
		RunE:  runDump,
	}

	rootCmd.AddCommand(dumpCmd)

	dumpCmd.Flags().String("format", stuber.DumpFormatYAML, "Output format: yaml, json")
	dumpCmd.Flags().StringP("output", "o", "stubs_export", "Output directory")
	dumpCmd.Flags().String("scheme", "http", "URL scheme: http or https")
	dumpCmd.Flags().String("source", "", "Filter by source (rest, mcp, proxy; default: all except file)")
}

func runDump(cmd *cobra.Command, _ []string) error {
	cfg := config.Load()

	format, _ := cmd.Flags().GetString("format")
	outDir, _ := cmd.Flags().GetString("output")
	filterSrc, _ := cmd.Flags().GetString("source")
	scheme, _ := cmd.Flags().GetString("scheme")

	if err := stuber.ValidateDumpSource(filterSrc); err != nil {
		return err
	}

	if err := stuber.ValidateDumpFormat(format); err != nil {
		return err
	}

	if scheme != "http" && scheme != "https" {
		return errors.Newf("unsupported scheme %q, use http or https", scheme)
	}

	endpoint := scheme + "://" + cfg.HTTPAddr

	stubs, err := fetchStubs(cmd.Context(), endpoint, filterSrc)
	if err != nil {
		return errors.Wrap(err, "fetch")
	}

	stubs = stuber.FilterForDump(stubs, filterSrc)
	if len(stubs) == 0 {
		cmd.Println("no stubs found")

		return nil
	}

	filesCount, err := stuber.DumpToDir(outDir, stubs, format)
	if err != nil {
		return err
	}

	cmd.Printf("\ntotal: %d files, %d stubs\n", filesCount, len(stubs))

	return nil
}

func fetchStubs(ctx context.Context, baseURL string, source string) ([]*stuber.Stub, error) {
	endpoint := baseURL + "/api/stubs"
	if source != "" {
		endpoint += "?source=" + source
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrapf(ErrUnexpectedStatus, "status: %s", resp.Status)
	}

	var payload []*stuber.Stub

	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()

	if err = decoder.Decode(&payload); err != nil {
		return nil, err
	}

	return payload, nil
}
