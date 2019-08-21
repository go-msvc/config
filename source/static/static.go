package static

import (
	"fmt"
	"strings"

	"github.com/go-msvc/config"
)

type source struct {
	name string
	data config.IData
	sub  *source
}

//New ...
func New(name string, data config.IData) config.ISource {
	return new(name, data)
}

func new(name string, data config.IData) *source {
	//skip leading '.'
	for len(name) > 0 && name[0] == '.' {
		name = name[1:]
	}

	//split name on first '.'
	parts := strings.SplitN(name, ".", 2)
	switch len(parts) {
	case 1:
		s := &source{
			name: name,
			data: data,
			sub:  nil,
		}
		return s
	case 2:
		s := &source{
			name: parts[0],
			data: nil,
			sub:  new(parts[1], data),
		}
		return s
	default:
	} //switch(nr parts in name)
	panic(fmt.Sprintf("New(\"%s\") -> %d parts", name, len(parts)))
} //New()

func (static source) Name() string {
	return "static(" + static.name + ")"
}

func (static source) Get(name string, tmpl config.IData) (config.IData, error) {
	//skip leading '.'
	for len(name) > 0 && name[0] == '.' {
		name = name[1:]
	}

	//split name on first '.'
	parts := strings.SplitN(name, ".", 2)
	switch len(parts) {
	case 1:
		if parts[0] == static.name {
			return static.data, nil
		}
		return nil, nil
	case 2:
		if parts[0] == static.name && static.sub != nil {
			return static.sub.Get(parts[1], tmpl)
		}
		return nil, nil
	default:
	} //switch(nr parts in name)
	panic(fmt.Sprintf("Get(\"%s\") -> %d parts", name, len(parts)))
}

func (static source) GetAll(name string) map[string]config.IData {
	//skip leading '.'
	for len(name) > 0 && name[0] == '.' {
		name = name[1:]
	}

	//split name on first '.'
	parts := strings.SplitN(name, ".", 2)
	switch len(parts) {
	case 1:
		if parts[0] == static.name && static.sub != nil {
			return map[string]config.IData{
				static.sub.name: static.sub.data,
			}
		}
		return nil
	case 2:
		if parts[0] == static.name && static.sub != nil {
			return static.sub.GetAll(parts[1])
		}
		return nil
	default:
	} //switch(nr parts in name)
	panic(fmt.Sprintf("GetAll(\"%s\") -> %d parts", name, len(parts)))
}
