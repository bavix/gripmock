package stuber

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-yaml"
)

const (
	DumpFormatYAML = "yaml"
	DumpFormatJSON = "json"
	dumpDirPerm    = 0o750
)

var (
	ErrUnknownDumpSource = errors.New("unknown dump source")
	ErrUnknownDumpFormat = errors.New("unknown dump format")
)

func ValidateDumpSource(source string) error {
	if source == "" {
		return nil
	}

	if IsKnownSource(source) {
		return nil
	}

	return errors.Wrapf(ErrUnknownDumpSource, "source: %q", source)
}

func ValidateDumpFormat(format string) error {
	if format == DumpFormatYAML || format == DumpFormatJSON {
		return nil
	}

	return errors.Wrapf(ErrUnknownDumpFormat, "format: %q", format)
}

func FilterForDump(stubs []*Stub, source string) []*Stub {
	filtered := make([]*Stub, 0, len(stubs))

	for stub := range slices.Values(stubs) {
		if stub == nil {
			continue
		}

		if source != "" {
			if stub.Source == source {
				filtered = append(filtered, stub)
			}

			continue
		}

		if stub.Source != SourceFile {
			filtered = append(filtered, stub)
		}
	}

	return filtered
}

func DumpFileKey(stub *Stub) string {
	if stub == nil {
		return "unknown"
	}

	return sanitizeDumpFileName(stub.Service + "_" + stub.Method)
}

func SortForDump(stubs []*Stub) {
	sort.Slice(stubs, func(i, j int) bool {
		leftKey := DumpFileKey(stubs[i])
		rightKey := DumpFileKey(stubs[j])

		if leftKey != rightKey {
			return leftKey < rightKey
		}

		if stubs[i].Service != stubs[j].Service {
			return stubs[i].Service < stubs[j].Service
		}

		return stubs[i].Method < stubs[j].Method
	})
}

func DumpToDir(outDir string, stubs []*Stub, format string) (int, error) {
	if err := os.MkdirAll(outDir, dumpDirPerm); err != nil {
		return 0, err
	}

	SortForDump(stubs)

	filesCount := 0

	for i := 0; i < len(stubs); {
		key := DumpFileKey(stubs[i])
		j := i + 1

		for j < len(stubs) && DumpFileKey(stubs[j]) == key {
			j++
		}

		filename := filepath.Join(outDir, key+"."+format)

		//nolint:gosec // G304: filename is derived from sanitized key and configured output directory.
		file, err := os.Create(filename)
		if err != nil {
			return filesCount, err
		}

		err = WriteDump(file, stubs[i:j], format)
		closeErr := file.Close()

		if err != nil {
			return filesCount, err
		}

		if closeErr != nil {
			return filesCount, closeErr
		}

		filesCount++
		i = j
	}

	return filesCount, nil
}

func WriteDump(writer io.Writer, stubs []*Stub, format string) error {
	type dumpMeta struct {
		Source string `json:"source,omitempty" yaml:"source,omitempty"`
	}

	type dumpRecord struct {
		Service string      `json:"service"          yaml:"service"`
		Method  string      `json:"method"           yaml:"method"`
		Input   InputData   `json:"input"            yaml:"input"`
		Inputs  []InputData `json:"inputs,omitempty" yaml:"inputs,omitempty"`
		Output  Output      `json:"output"           yaml:"output"`
		Headers InputHeader `json:"headers"          yaml:"headers"`
		Meta    *dumpMeta   `json:"_meta,omitempty"  yaml:"_meta,omitempty"` //nolint:tagliatelle
	}

	data := make([]dumpRecord, 0, len(stubs))
	for stub := range slices.Values(stubs) {
		if stub == nil {
			continue
		}

		rec := dumpRecord{
			Service: stub.Service,
			Method:  stub.Method,
			Input:   stub.Input,
			Inputs:  stub.Inputs,
			Output:  stub.Output,
			Headers: stub.Headers,
		}

		if stub.Source != "" {
			rec.Meta = &dumpMeta{Source: stub.Source}
		}

		data = append(data, rec)
	}

	if format == DumpFormatJSON {
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")

		return encoder.Encode(data)
	}

	encoder := yaml.NewEncoder(writer)

	if err := encoder.Encode(data); err != nil {
		_ = encoder.Close()

		return err
	}

	return encoder.Close()
}

func sanitizeDumpFileName(name string) string {
	value := strings.ReplaceAll(name, ".", "_")
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, ":", "_")

	return value
}
