package yaml2json

import (
	"github.com/goccy/go-yaml"

	"github.com/bavix/gripmock/v3/pkg/plugins"
)

// Convertor provides functionality for converting YAML data to JSON format.
type Convertor struct {
	engine *engine
}

// New creates a new Convertor instance.
func New(reg plugins.Registry) *Convertor {
	return &Convertor{engine: newEngine(reg)}
}

// Execute executes the given YAML data and returns the JSON representation.
//
// It takes a name and data as input parameters.
// The name parameter is used as a reference for the execution.
// The data parameter is the YAML data to be executed.
//
// It returns a byte slice and an error.
// The byte slice contains the JSON representation of the executed YAML data.
// The error is non-nil if there was an error during the execution.
func (t *Convertor) Execute(name string, data []byte) ([]byte, error) {
	jsonData, err := t.engine.Execute(name, data)
	if err != nil {
		return nil, err
	}

	return yaml.YAMLToJSON(jsonData) //nolint:wrapcheck
}
