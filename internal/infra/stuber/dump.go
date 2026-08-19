package stuber

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-yaml"
	"github.com/google/uuid"
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
	records := make([]map[string]any, 0, len(stubs))

	for stub := range slices.Values(stubs) {
		if stub == nil {
			continue
		}

		records = append(records, dumpRecordOf(stub))
	}

	if format == DumpFormatJSON {
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")

		return encoder.Encode(records)
	}

	encoder := yaml.NewEncoder(writer)

	if err := encoder.Encode(records); err != nil {
		_ = encoder.Close()

		return err
	}

	return encoder.Close()
}

func dumpRecordOf(stub *Stub) map[string]any {
	record := map[string]any{
		"service": stub.Service,
		"method":  stub.Method,
		"output":  dumpOutput(stub.Output),
	}

	addDumpScalars(record, stub)
	addDumpMatchers(record, stub)

	if len(stub.Effects) > 0 {
		record["effects"] = stub.Effects
	}

	if stub.Source != "" {
		record["_meta"] = map[string]any{"source": stub.Source}
	}

	return record
}

func dumpOutput(output Output) map[string]any {
	out := pruneNulls(structToMap(output))

	if output.Code == nil {
		if msg, ok := out["error"].(string); ok && msg == "" {
			delete(out, "error")
		}
	}

	return out
}

func addDumpScalars(record map[string]any, stub *Stub) {
	if stub.ID != uuid.Nil {
		record["id"] = stub.ID.String()
	}

	if stub.Session != "" {
		record["session"] = stub.Session
	}

	if stub.Priority != 0 {
		record["priority"] = stub.Priority
	}

	if stub.Options.Times != 0 {
		record["options"] = map[string]any{"times": stub.Options.Times}
	}
}

func addDumpMatchers(record map[string]any, stub *Stub) {
	if headers := pruneNulls(structToMap(stub.Headers)); len(headers) > 0 {
		record["headers"] = headers
	}

	if input := pruneNulls(structToMap(stub.Input)); len(input) > 0 {
		record["input"] = input
	}

	if inputs := dumpInputs(stub.Inputs); len(inputs) > 0 {
		record["inputs"] = inputs
	}
}

func dumpInputs(inputs []InputData) []map[string]any {
	out := make([]map[string]any, 0, len(inputs))
	for _, in := range inputs {
		out = append(out, pruneNulls(structToMap(in)))
	}

	return out
}

func structToMap(v any) map[string]any {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var out map[string]any
	if err := decoder.Decode(&out); err != nil {
		return nil
	}

	normalized, _ := normalizeNumbers(out).(map[string]any)

	return normalized
}

func narrowNumber(value json.Number) any {
	if integer, err := value.Int64(); err == nil {
		return integer
	}

	if unsigned, err := strconv.ParseUint(value.String(), 10, 64); err == nil {
		return unsigned
	}

	if float, err := value.Float64(); err == nil {
		return float
	}

	return value.String()
}

func normalizeNumbers(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			typed[key] = normalizeNumbers(nested)
		}

		return typed
	case []any:
		for i, nested := range typed {
			typed[i] = normalizeNumbers(nested)
		}

		return typed
	case json.Number:
		return narrowNumber(typed)
	default:
		return value
	}
}

func pruneNulls(m map[string]any) map[string]any {
	for key, value := range m {
		if value == nil {
			delete(m, key)
		}
	}

	return m
}

func sanitizeDumpFileName(name string) string {
	value := strings.ReplaceAll(name, ".", "_")
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, ":", "_")

	return value
}
