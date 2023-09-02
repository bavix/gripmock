package yaml2json

import (
	"github.com/goccy/go-yaml"
)

type Convertor struct {
	engine *engine
}

func New() *Convertor {
	return &Convertor{engine: &engine{}}
}

func (t *Convertor) Execute(name string, data []byte) ([]byte, error) {
	bytes, err := t.engine.Execute(name, data)
	if err != nil {
		return nil, err
	}

	return yaml.YAMLToJSON(bytes)
}
