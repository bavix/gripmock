package yaml2json

import (
	"github.com/goccy/go-yaml"
	"github.com/tokopedia/gripmock/pkg/template"
)

type Convertor struct {
	engine *template.Engine
}

func New() *Convertor {
	return &Convertor{engine: template.New()}
}

func (t *Convertor) Execute(name string, data []byte) ([]byte, error) {
	bytes, err := t.engine.Execute(name, data)
	if err != nil {
		return nil, err
	}

	return yaml.YAMLToJSON(bytes)
}
