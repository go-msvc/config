package static

import (
	"github.com/go-msvc/config"
	"github.com/go-msvc/logger"
)

var log = logger.ForThisPackage()

type source struct {
	name  string
	value interface{}
}

//New ...
func New(name string, value interface{}) config.ISource {
	return source{
		name:  name,
		value: value,
	}
}

func (static source) Name() string {
	return "static(" + static.name + ")"
}

func (static source) Get(name string) (interface{}, error) {
	if name == static.name {
		return static.value, nil
	}
	return nil, nil
}

// func new(name string, value interface{}) *source {
// 	//skip leading '.'
// 	for len(name) > 0 && name[0] == '.' {
// 		name = name[1:]
// 	}
// 	log.Debugf("new static config name=\"%s\"", name)

// 	//split name on first '.'
// 	parts := strings.SplitN(name, ".", 2)
// 	switch len(parts) {
// 	case 1:
// 		s := &source{
// 			name:  name,
// 			value: value,
// 			sub:   nil,
// 		}
// 		log.Debugf("s=%T:%v", s, s)
// 		return s
// 	case 2:
// 		s := &source{
// 			name:  parts[0],
// 			value: value,
// 			sub:   new(parts[1], data),
// 		}
// 		log.Debugf("s=%T:%v", s, s)
// 		return s
// 	default:
// 	} //switch(nr parts in name)
// 	panic(fmt.Sprintf("New(\"%s\") -> %d parts", name, len(parts)))
// } //New()

// func (static source) Name() string {
// 	return "static(" + static.name + ")"
// }

// func (static source) Get(name string) (interface{}, error) {
// 	log.Debugf("static.Get(%s)", name)
// 	//skip leading '.'
// 	for len(name) > 0 && name[0] == '.' {
// 		name = name[1:]
// 	}

// 	//split name on first '.'
// 	parts := strings.SplitN(name, ".", 2)
// 	switch len(parts) {
// 	case 1:
// 		if parts[0] == static.name {
// 			//found, print to JSON then parse into the specified tmpl
// 			jsonData, _ := json.Marshal(static.data)
// 			outPtrValue := reflect.New(reflect.TypeOf(tmpl))
// 			if err := json.Unmarshal(jsonData, outPtrValue.Interface()); err != nil {
// 				return nil, fmt.Errorf("failed to decode %s into %T", name, tmpl)
// 			}
// 			cfgData := outPtrValue.Elem().Interface().(config.IData)
// 			if err := cfgData.Validate(); err != nil {
// 				return nil, errors.Wrapf(err, "invalid %s", name)
// 			}
// 			return outPtrValue.Elem().Interface().(config.IData), nil
// 		}
// 		return nil, nil
// 	case 2:
// 		if parts[0] == static.name && static.sub != nil {
// 			return static.sub.Get(parts[1], tmpl)
// 		}
// 		return nil, nil
// 	default:
// 	} //switch(nr parts in name)
// 	panic(fmt.Sprintf("Get(\"%s\") -> %d parts", name, len(parts)))
// }

// func (static source) GetAll(name string) map[string]interface{} {
// 	//skip leading '.'
// 	for len(name) > 0 && name[0] == '.' {
// 		name = name[1:]
// 	}
// 	log.Debugf("GetAll(%s)", name)

// 	//split name on first '.'
// 	parts := strings.SplitN(name, ".", 2)
// 	switch len(parts) {
// 	case 1:
// 		if parts[0] == static.name && static.sub != nil {
// 			log.Debugf("Return: [%s]:%v", static.sub.name, static.sub.data)
// 			return map[string]interface{}{
// 				static.sub.name: static.sub.data,
// 			}
// 		}
// 		return nil
// 	case 2:
// 		if parts[0] == static.name && static.sub != nil {
// 			return static.sub.GetAll(parts[1])
// 		}
// 		log.Debugf("Return: nil")
// 		return nil
// 	default:
// 	} //switch(nr parts in name)
// 	panic(fmt.Sprintf("GetAll(\"%s\") -> %d parts", name, len(parts)))
// }
