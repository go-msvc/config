package config

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/go-msvc/errors"
	"github.com/go-msvc/logger"
)

var log = logger.ForThisPackage()

const ConstructorMethodName = "Create"

//IValidator ...
type IValidator interface {
	Validate() error
}

type IConfigurable interface {
	Current() interface{}
}

func MustAdd(defaultStructValue interface{}) IConfigurable {
	if c, err := Add(defaultStructValue); err != nil {
		panic(fmt.Sprintf("%+v", err))
	} else {
		return c
	}
}

//add struct with configurable fields and default values
func Add(defaultStructValue interface{}) (IConfigurable, error) {
	if defaultStructValue == nil {
		return nil, errors.Errorf("Add(nil)")
	}
	t := reflect.TypeOf(defaultStructValue)
	if err := validateConfigStructType(t); err != nil {
		return nil, errors.Wrapf(err, "cannot use %T as config struct", defaultStructValue)
	}

	//create new configurable and do init load
	c := &configurable{
		structType:         t,
		defaultStructValue: reflect.ValueOf(defaultStructValue),
	}
	if err := c.load(); err != nil {
		return nil, errors.Wrapf(err, "failed on init load")
	}

	//add to list
	mutex.Lock()
	defer mutex.Unlock()
	configurableList = append(configurableList, c)

	return c, nil
} //Add()

//check that the specified type is a struct containing only configurable fields
func validateConfigStructType(t reflect.Type) error {
	if t.Kind() != reflect.Struct {
		return errors.Errorf("%s is not a struct", t.Name())
	}
	if t.NumField() == 0 {
		return errors.Errorf("%s has no configurable fields", t.Name())
	}
	tempPtrValue := reflect.New(t)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tagName := f.Tag.Get("json")
		if tagName == "" {
			return errors.Errorf("%s.%s has no json tag", t.Name(), f.Name)
		}
		vf := tempPtrValue.Elem().Field(i)
		if !vf.CanInterface() {
			return errors.Errorf("%s.%s is private", t.Name(), f.Name)
		}

		//if field type has Constructor - check that it can be used
		if constructorMethod, hasConstructor := f.Type.MethodByName(ConstructorMethodName); hasConstructor {
			if constructorMethod.Type.NumIn() != 2 { //2 = receiver + 1 arg
				return errors.Errorf("%s.%s: %s.%s() must take 1 argument for configurable value", t.Name(), f.Name, f.Type.Name(), ConstructorMethodName)
			}
			if constructorMethod.Type.In(1).Kind() == reflect.Ptr {
				return errors.Errorf("%s.%s: %s.%s() may not take pointer argument", t.Name(), f.Name, f.Type.Name(), ConstructorMethodName)
			}
			if constructorMethod.Type.NumOut() != 2 {
				return errors.Errorf("%s.%s: %s.%s() does not return (%s,error)", t.Name(), f.Name, f.Type.Name(), ConstructorMethodName, f.Type.Name())
			}
			if constructorMethod.Type.Out(0) != f.Type {
				return errors.Errorf("%s.%s: %s.%s() return (%s,%s) instead of (%s,error)", t.Name(), f.Name, f.Type.Name(), ConstructorMethodName, constructorMethod.Type.Out(0).Name(), constructorMethod.Type.Out(1).Name(), f.Type.Name())
			}
			if constructorMethod.Type.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
				return errors.Errorf("%s.%s: %s.%s() does not return (%s,error)", t.Name(), f.Name, f.Type.Name(), ConstructorMethodName, f.Type.Name())
			}
		} //if has Constructor() method
	} //for each field
	return nil
} //validateConfigStructType()

var (
	mutex            sync.Mutex
	configurableList = make([]*configurable, 0)
)

type configurable struct {
	structType         reflect.Type
	defaultStructValue reflect.Value
	structPtrValue     reflect.Value
	valueTimestamp     time.Time
}

func (c *configurable) Current() interface{} {
	return c.structPtrValue.Elem().Interface()
}

func (c *configurable) load() error {
	//allocate a new copy of the struct
	structPtrValue := reflect.New(c.structType)
	count := 0
	for i := 0; i < c.structType.NumField(); i++ {
		f := c.structType.Field(i)
		tagName := f.Tag.Get("json")

		//get field value in the new struct
		vf := structPtrValue.Elem().Field(i)

		//check if just a value or a constructed
		if configureMethod, hasConstructor := f.Type.MethodByName(ConstructorMethodName); !hasConstructor {
			//config value without constructor: get it and store in struct field
			log.Debugf("%s.%s type %s has no %s()", c.structType.Name(), f.Name, f.Type.Name(), ConstructorMethodName)
			configuredFieldValue, err := GetAndWatch(tagName, c.defaultStructValue.Field(i).Interface(), c)
			if err != nil {
				return errors.Wrapf(err, "cannot get config for %s.%s", c.structType.Name(), f.Name)
			}
			vf.Set(reflect.ValueOf(configuredFieldValue))
			log.Debugf("%s = (%T) %+v", tagName, configuredFieldValue, configuredFieldValue)
		} else {
			//config value with constructor: get the constructor arg from config
			configuredValue, err := GetAndWatch(tagName, reflect.New(configureMethod.Type.In(1)).Elem().Interface(), c)
			if err != nil {
				return errors.Wrapf(err, "config error on %s", tagName)
			}
			log.Debugf("config Value := (%T) %+v", configuredValue, configuredValue)

			//if not configured, item remains nil
			//if has configured value, then call Configure(value)
			if configuredValue == nil {
				if f.Type.Kind() != reflect.Ptr {
					return errors.Errorf("config required for %s", tagName)
				}
				log.Debugf("%s.%s (optional) is not configured", c.structType.Name(), f.Name)
			} else {
				if f.Type.Kind() == reflect.Ptr {
					//field is a pointer to type with constructor:
					//we need to allocate it before we can call the constructor
					vf.Set(reflect.New(f.Type.Elem()))
				}
				//prepare constructor arguments
				args := []reflect.Value{
					reflect.ValueOf(configuredValue),
				}

				//call the constructor
				log.Debugf("%s.%s: calling %s.%s() ...", c.structType.Name(), f.Name, f.Type.Name(), ConstructorMethodName)
				log.Debugf("vf = %v", vf.Type().Name())
				result := vf.MethodByName(ConstructorMethodName).Call(args)

				//fail on error (2nd result value)
				if !result[1].IsNil() {
					return errors.Wrapf(result[1].Interface().(error), "%s.%s: %s.%s() failed", c.structType.Name(), f.Name, f.Type.Name(), ConstructorMethodName)
				}

				//constructor succeeded
				if f.Type.Kind() == reflect.Ptr {
					if !result[0].Type().AssignableTo(f.Type.Elem()) {
						return errors.Errorf("%s.%s: %s.%s() returns incompatible type %v", c.structType.Name(), f.Name, f.Type.Elem().Name(), ConstructorMethodName, result[0].Type().Name())
					}
					vf.Elem().Set(result[0])
				} else {
					if !result[0].Type().AssignableTo(f.Type) {
						return errors.Errorf("%s.%s: %s.%s() returns incompatible type %v", c.structType.Name(), f.Name, f.Type.Name(), ConstructorMethodName, result[0].Type().Name())
					}
					vf.Set(result[0])
				}
				log.Debugf("%s Configure(%+v) success", tagName, configuredValue)
			} //if configured
		} //if has Configure() method
		count++
	} //for each field

	//loaded/constrcuted all fields from config
	//if config struct implements IValidator, call it now
	if validator, ok := structPtrValue.Interface().(IValidator); ok {
		if err := validator.Validate(); err != nil {
			return errors.Wrapf(err, "invalid config")
		}
	}

	c.structPtrValue = structPtrValue
	c.valueTimestamp = time.Now()
	log.Debugf("Successfully loaded %s", c.structType.Name())
	return nil
} //configurable.load()

func (c *configurable) Notify(name string) {
	log.Debugf("%v: NOTIFIER(%s)", c.structType.Name(), name)
	if err := c.load(); err != nil {
		log.Errorf("failed to load change to %s", name)
	} else {
		log.Debugf("loaded change to %s", name)
	}
}
