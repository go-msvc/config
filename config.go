package config

import (
	"reflect"

	"github.com/go-msvc/errors"
	"github.com/go-msvc/logger"
)

var log = logger.New().WithLevel(logger.LevelDebug)

type Constructor interface {
	Create() (any, error)
}

// indicate that your module require a conigurable item
// call this before the config is loaded, i.e. in your package's init() func
// because config is loaded at the start of main()
func Require(name string, tmpl interface{}) error {
	if valueByName != nil {
		return errors.Errorf("config.Require(%s) after config was loaded", name)
	}
	if existingTmpl, ok := required[name]; ok {
		if reflect.TypeOf(tmpl) != reflect.TypeOf(existingTmpl) {
			return errors.Errorf("config.Require(%s) with conflicting type %v != %v already required", name, reflect.TypeOf(tmpl), reflect.TypeOf(existingTmpl))
		}
	} else {
		required[name] = tmpl
	}
	return nil
}

var (
	required      = map[string]interface{}{}
	valueByName   map[string]interface{} //validated value from source
	createdByName map[string]interface{} //constructor for value.Create()
)

// load config from all config sources
// after this, sources cannot be added and Required will also fail
// to ensure that documentation can be generated
func Load() error {
	if valueByName == nil {
		//get each required item from all sources
		//each item must be configured and only in one of the sources
		valueByName = map[string]interface{}{}
		for configName, requiredTmpl := range required {
			foundInSoureNames := map[string]interface{}{}
			for sourceName, source := range sources {
				configuredValue, err := source.GetInto(configName, requiredTmpl)
				if err != nil {
					return errors.Wrapf(err, "source(%s).config(%s) is invalid", sourceName, configName)
				}
				if configuredValue != nil {
					valueByName[configName] = configuredValue
					foundInSoureNames[sourceName] = configuredValue
					log.Debugf("Source(%s).Configured(%s): %T", sourceName, configName, configuredValue)
				}
			}
			if len(foundInSoureNames) != 1 {
				return errors.Errorf("%s found in %d source", configName, len(foundInSoureNames))
			}
		} //for each required config

		//construct all required items where config is a constructor
		//do this after above loop so above loop will first fail on any missing config
		//and not one some construction error before all config was loaded
		for configName, configValue := range valueByName {
			if constructor, ok := configValue.(Constructor); ok {
				created, err := constructor.Create()
				if err != nil {
					return errors.Wrapf(err, "failed to construct(%s)", configName)
				}
				createdByName[configName] = created
				log.Debugf("Constructed(%s): %T", configName, created)
			}
		}
	} //if not already loaded
	return nil
} //Load()

// Get an item that you specified with Require()
// If config was not yet loaded, this will load config
// and config can only be loaded once, and Require()
// will fail after Load() was called.
func Get(name string) (any, error) {
	if err := Load(); err != nil {
		return nil, errors.Errorf("Get(%s) failed because Load failed", name)
	}
	if _, ok := required[name]; !ok {
		return nil, errors.Errorf("config(%s) not required", name)
	}
	if v, ok := createdByName[name]; ok {
		return v, nil
	}
	if v, ok := valueByName[name]; ok {
		return v, nil
	}
	panic("not expected to get here")
} //Get()
