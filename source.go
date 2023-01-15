package config

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/go-msvc/data"
	"github.com/go-msvc/errors"
)

type Source interface {
	GetInto(name string, tmpl interface{}) (interface{}, error)
}

type namedSource struct {
	name   string
	source Source
}

var (
	sources = []namedSource{}
)

// todo: provide mechanism to write config set to a backup, for audit
// but also for use when sources cannot be reached.

// sources are used in order of being added
// call this in main func, not in init(), as that will not allow
// control of the order of sources, and the order determine which
// one is used, so you can have multiple sources for redundancy,
// e.g. read remote, but make local copy so if restart and remote
// is not reachable, then read local backup, or just build that
// into your source if you only have one
func AddSource(name string, source Source) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.Errorf("invalid config source name \"%s\"", name)
	}
	if source == nil {
		return errors.Errorf("cannot add config source nil")
	}
	sources = append(sources, namedSource{name: name, source: source})
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
