package config

import (
	"encoding/json"
	"reflect"
	"sync"

	"github.com/go-msvc/errors"
)

var (
	setMutex  sync.Mutex
	setValues = map[string]interface{}{}
)

//Sources of config
func Sources() ISources {
	return allSources
}

//Get configured value
//return error if source failed or value is invalid
func Get(name string, defaultValue interface{}) (interface{}, error) {
	return allSources.Get(name, defaultValue, nil)
}

//Same as Get but get notifier if the value changes in the source it was obtained from
//todo: also wants to get notifier when optional values becomes available in any source...
func GetAndWatch(name string, defaultValue interface{}, notifier INotifier) (interface{}, error) {
	return allSources.Get(name, defaultValue, notifier)
}

//ISources is a collection of config sources
type ISources interface {
	Reset()        //removes all sources
	Add(s ISource) //add in order of execution
	Get(name string, defaultValue interface{}, notifier INotifier) (interface{}, error)
}

var (
	allSources ISources
)

func init() {
	allSources = &sources{
		list: make([]ISource, 0),
	}
}

type sources struct {
	mutex sync.Mutex
	list  []ISource
}

func (s *sources) Reset() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.list = []ISource{}
	log.Infof("debug all config sources")
}

//Add to default list of config sources used by Get(name)
func (s *sources) Add(source ISource) {
	log.Debugf("sources(%p).Add(%p)", s, source)
	if s == nil || source == nil {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, existing := range s.list {
		if source == existing {
			return
		}
	}
	s.list = append(s.list, source)
} //sources.Add()

//return:
//		(value,nil) if found + valid,
//		(nil,nil)   if not defined in any source
//		(nil,error) if any source returned an error
func (s *sources) Get(name string, defaultValue interface{}, notifier INotifier) (interface{}, error) {
	log.Debugf("sources(%p).Get(%s,%T)", s, name, defaultValue)
	value, err := s.getValue(name, defaultValue, notifier)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get(%s)", name)
	}
	if value == nil {
		//todo: if has notifier, watch when it becomes available? or rather do that on different call like Wait() because
		//caller might not expect notifier reference to be kept open when value is not configured...
		return nil, nil //not configured and default is nil
	}

	log.Debugf("got %s: (%T) %+v", name, value, value)

	//type check can only be applied if default was specified
	if defaultValue != nil {
		t := reflect.TypeOf(defaultValue)

		//do type conversion if necessary
		if reflect.TypeOf(value) != reflect.TypeOf(defaultValue) {
			log.Debugf("convert %T -> %T ...", value, defaultValue)

			jsonValue, err := json.Marshal(value)
			if err != nil {
				return nil, errors.Wrapf(err, "cannot write value as JSON")
			}
			newValuePtr := reflect.New(t)
			if err := json.Unmarshal(jsonValue, newValuePtr.Interface()); err != nil {
				return nil, errors.Wrapf(err, "cannot parse JSON value")
			}
			value = newValuePtr.Elem().Interface()
			log.Debugf("parsed %s: (%T) %+v", name, value, value)
		}

		if validator, ok := value.(IValidator); ok {
			if err := validator.Validate(); err != nil {
				return nil, errors.Wrapf(err, "invalid %s", name)
			}
			log.Debugf("validated %s: (%T) %+v", name, value, value)
		}
	}
	log.Debugf("use %s: (%T) %+v", name, value, value)
	return value, nil
} //sources.Get()

//just get the value from sources, no validation yet
func (s *sources) getValue(name string, defaultValue interface{}, notifier INotifier) (interface{}, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, source := range s.list {
		value, err := source.Get(name, notifier)
		if err != nil {
			return nil, errors.Wrapf(err, "source(%s).Get(%s) failed", source.Name(), name)
		}
		if value != nil {
			log.Debugf("source(%s).get(%s) -> (%T) %+v", source.Name(), name, value, value)
			return value, nil
		}
	} //for each source
	return defaultValue, nil
} //sources.getValue()
