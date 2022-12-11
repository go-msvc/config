package config

import (
	"encoding/json"
	"os"

	"github.com/go-msvc/data"
	"github.com/go-msvc/errors"
)

type Source interface {
	GetInto(name string, tmpl interface{}) (interface{}, error)
}

var (
	sources = map[string]Source{}
)

func AddSource(name string, source Source) error {
	//todo: check
	sources[name] = source
	return nil
}

// defaultfile is used if config is loaded with no sources
// to load config from file "./config.json"
type defaultfile struct {
	value interface{}
}

func (f defaultfile) GetInto(name string, tmpl interface{}) (interface{}, error) {
	if f.value == nil {
		fn := "./config.json"
		cf, err := os.Open(fn)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot open file %s", fn)
		}
		defer cf.Close()
		if err := json.NewDecoder(cf).Decode(&f.value); err != nil {
			return nil, errors.Wrapf(err, "failed to decode JSON from file %s", fn)
		}
	}
	return data.GetInto(f.value, name, tmpl)
}
