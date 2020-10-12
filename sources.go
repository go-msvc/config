package config

import (
	"encoding/json"
	"reflect"
	"sync"

	"github.com/go-msvc/errors"
	logger "github.com/go-msvc/logger"
)

var log = logger.ForThisPackage()

//Set a config value to override any other source
//this is like setting hard coded value, but the code can
//change it at any time, but no sources will be queried once
//this is set. Set with value nil to undo the Set()
func Set(name string, value interface{}) {
	setMutex.Lock()
	defer setMutex.Unlock()
	if value == nil {
		delete(setValues, name)
		log.Debugf("Del %s", name)
	} else {
		setValues[name] = value
		log.Debugf("Set %s: (%T) %+v", name, value, value)
	}
}

func get(name string) (interface{}, bool) {
	setMutex.Lock()
	defer setMutex.Unlock()
	if value, ok := setValues[name]; ok {
		log.Debugf("Got: %s (%T) %+v", name, value, value)
		return value, true
	}
	return nil, false
}

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
	return allSources.Get(name, defaultValue)
}

//ISources is a collection of config sources
type ISources interface {
	Reset()        //removes all sources
	Add(s ISource) //add in order of execution
	Get(name string, defaultValue interface{}) (interface{}, error)
	//GetAll(name string) map[string]interface{}
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

//return nil,nil if not found, data,nil if found + valid, nil,error if invalid or can't get
func (s *sources) Get(name string, defaultValue interface{}) (interface{}, error) {
	log.Debugf("sources(%p).Get(%s,%T)", s, name, defaultValue)
	value := defaultValue
	if setValue, isSet := get(name); isSet {
		value = setValue
		log.Debugf("use %s: (%T) %+v", name, value, value)
	} else {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		for _, source := range s.list {
			sourceValue, err := source.Get(name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed in source(%s).Get(%s)", source.Name(), name)
			}
			if sourceValue != nil {
				value = sourceValue
				break
			}
		} //for each source
	} //if !isSet

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
	log.Debugf("return %s: (%T) %+v", name, value, value)
	return value, nil
} //Get()

// func (s *sources) GetAll(name string) map[string]interface{} {
// 	s.mutex.Lock()
// 	defer s.mutex.Unlock()

// 	all := make(map[string]interface{})
// 	for _, source := range s.list {
// 		if sourceList := source.GetAll(name); len(sourceList) > 0 {
// 			//merge list into all
// 			for n, d := range sourceList {
// 				if _, ok := all[n]; ok {
// 					log.Errorf("Ignore duplicate config %s:%s", source.Name(), n)
// 				} else {
// 					all[n] = d
// 				}
// 			}
// 		}
// 	}
// 	log.Debugf("Found %d %s.*", len(all), name)
// 	for n, v := range all {
// 		log.Debugf("  [%s]:(%T):%v", n, v, v)
// 	}
// 	return all
// } //sources.GetAll()
