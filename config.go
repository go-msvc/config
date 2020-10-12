package config

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/go-msvc/errors"
	"github.com/go-msvc/logger"
)

const (
	ConstructorMethodName = "Create"
	DestructorMethodName  = "Destroy"
)

//IValidator ...
type IValidator interface {
	Validate() error
}

type IConfigurable interface {
	//Current() returns the current value only if the struct has no constructed fields
	//because it is dangerouse to use constructed items that may be destroyed while you
	//are still using it.
	Current() (structValue interface{})

	//Use() returns the current value with a release() function to call when you're done
	Use() (structValue interface{}, release func())
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
	hasConstructedValues, err := validateConfigStructType(t)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot use %T as config struct", defaultStructValue)
	}

	//create new configurable and do init load
	c := &configurable{
		structType:           t,
		defaultStructValue:   reflect.ValueOf(defaultStructValue),
		hasConstructedValues: hasConstructedValues,
	}
	if err := c.load(nil); err != nil {
		return nil, errors.Wrapf(err, "failed on init load")
	}

	//add to list
	mutex.Lock()
	defer mutex.Unlock()
	configurableList = append(configurableList, c)

	return c, nil
} //Add()

//check that the specified type is a struct containing only configurable fields
func validateConfigStructType(t reflect.Type) (hasConstructedValues bool, err error) {
	if t.Kind() != reflect.Struct {
		return false, errors.Errorf("%s is not a struct", t.Name())
	}
	if t.NumField() == 0 {
		return false, errors.Errorf("%s has no configurable fields", t.Name())
	}
	hasConstructedValues = false
	tempPtrValue := reflect.New(t)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		valType := f.Type
		constructorMethod, hasConstructorMethod := valType.MethodByName(ConstructorMethodName)
		if hasConstructorMethod {
			log.Debugf("(%s).%s() = %v", valType, ConstructorMethodName, constructorMethod)
		}

		//optional := false
		if valType.Kind() == reflect.Ptr {
			//field is ptr, i.e. config is optional
			//optional = true
			valType = valType.Elem()
		}
		if valType.Kind() == reflect.Ptr {
			return false, errors.Errorf("%s.%s type may not be double pointer %v", t.Name(), f.Name, f.Type.Name())
		}

		tagName := f.Tag.Get("json")
		if tagName == "" {
			return false, errors.Errorf("%s.%s has no json tag", t.Name(), f.Name)
		}
		vf := tempPtrValue.Elem().Field(i)
		if !vf.CanInterface() {
			return false, errors.Errorf("%s.%s is private", t.Name(), f.Name)
		}

		//if field type has Constructor - check that it can be used
		if hasConstructorMethod {
			if constructorMethod.Type.NumIn() != 2 { //2 = receiver + 1 arg
				return false, errors.Errorf("%s.%s: %s.%s() must take 1 argument for configurable value", t.Name(), f.Name, f.Type.Name(), ConstructorMethodName)
			}
			// if constructorMethod.Type.In(0).Kind() == reflect.Ptr {
			// 	//constructor has ptr receiver
			// } else {
			// 	//constructor has value receiver
			// }
			if constructorMethod.Type.In(1).Kind() == reflect.Ptr {
				return false, errors.Errorf("%s.%s: %s.%s() may not take pointer argument", t.Name(), f.Name, f.Type.Name(), ConstructorMethodName)
			}
			if constructorMethod.Type.NumOut() != 2 {
				return false, errors.Errorf("%s.%s: %s.%s() does not return (%s,error)", t.Name(), f.Name, f.Type.Name(), ConstructorMethodName, f.Type.Name())
			}
			//constructor must return ptr to valType
			if constructorMethod.Type.Out(0).Kind() != reflect.Ptr || constructorMethod.Type.Out(0).Elem() != valType {
				return false, errors.Errorf("%s.%s: %s.%s() returns (%v,%s) instead of (*%v,error)", t.Name(), f.Name, f.Type, ConstructorMethodName, constructorMethod.Type.Out(0).Name(), constructorMethod.Type.Out(1).Name(), f.Type)
			}
			if constructorMethod.Type.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
				return false, errors.Errorf("%s.%s: %s.%s() does not return (%s,error)", t.Name(), f.Name, f.Type.Name(), ConstructorMethodName, f.Type.Name())
			}
			hasConstructedValues = true
		} //if has constructor

		//if field type has destructor - check that it can be used
		if destructorMethod, hasDestructor := f.Type.MethodByName(DestructorMethodName); hasDestructor {
			if destructorMethod.Type.NumIn() != 1 { //1 = receiver + 0 args
				return false, errors.Errorf("%s.%s: %s.%s() must take no arguments", t.Name(), f.Name, f.Type.Name(), DestructorMethodName)
			}
			if destructorMethod.Type.NumOut() != 0 {
				return false, errors.Errorf("%s.%s: %s.%s() should not return any values", t.Name(), f.Name, f.Type.Name(), DestructorMethodName)
			}
		} //if has destructor
	} //for each field
	return hasConstructedValues, nil
} //validateConfigStructType()

type configurable struct {
	//this never changes:
	structType           reflect.Type
	defaultStructValue   reflect.Value
	hasConstructedValues bool

	//this is defined each time config struct was loaded
	currentValue       *configValue
	currentReleaseFunc func()
}

func (c *configurable) Current() interface{} {
	if c.hasConstructedValues {
		panic(errors.Errorf("%T.Current() must be replaced with %T.Use()", c, c))
	}
	return c.currentValue.structPtrValue.Elem().Interface()
}

func (c *configurable) load(onlyNames []string) error {
	//allocate a new copy of the struct
	structPtrValue := reflect.New(c.structType)
	count := 0
	for i := 0; i < c.structType.NumField(); i++ {
		f := c.structType.Field(i)
		tagName := f.Tag.Get("json")

		//get field value in the new struct
		vf := structPtrValue.Elem().Field(i)

		//if filtered only some names, and name not in that list,
		//copy current value and don't reload/reconstruct
		//typically after a named value changed in config
		if c.currentValue != nil && len(onlyNames) > 0 && !stringInList(tagName, onlyNames) {
			log.Debugf("keep %s", tagName)
			vf.Set(c.currentValue.structPtrValue.Elem().Field(i))
			continue
		}

		//check if just a value or a constructed
		if constructorMethod, hasConstructor := f.Type.MethodByName(ConstructorMethodName); !hasConstructor {
			//config value without constructor: get it and store in struct field
			log.Debugf("%s.%s type %s has no %s()", c.structType.Name(), f.Name, f.Type.Name(), ConstructorMethodName)
			configuredFieldValue, err := GetAndWatch(tagName, c.defaultStructValue.Field(i).Interface(), c)
			if err != nil {
				return errors.Wrapf(err, "%s.%s: cannot get config %s", c.structType.Name(), f.Name, tagName)
			}
			if f.Type.Kind() != reflect.Ptr && configuredFieldValue == nil {
				//not optional and absent
				return errors.Wrapf(err, "%s.%s: missing config %s", c.structType.Name(), f.Name, tagName)
			}
			if configuredFieldValue != nil {
				vf.Set(reflect.ValueOf(configuredFieldValue))
				log.Debugf("%s.%s: %s = (%T) %+v", c.structType.Name(), f.Name, tagName, configuredFieldValue, configuredFieldValue)
			} else {
				log.Debugf("%s.%s: %s (optional) not configured", c.structType.Name(), f.Name, tagName)
			}
		} else {
			//config value with constructor: get the constructor arg from config
			configuredValue, err := GetAndWatch(tagName, reflect.New(constructorMethod.Type.In(1)).Elem().Interface(), c)
			if err != nil {
				return errors.Wrapf(err, "config error on %s", tagName)
			}
			log.Debugf("%s.%s: %s = (%T) %+v", c.structType.Name(), f.Name, tagName, configuredValue, configuredValue)

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
					if result[0].Type().AssignableTo(f.Type) {
						vf.Set(result[0])
					} else {
						return errors.Errorf("%s.%s: %s.%s() returns incompatible type %v", c.structType.Name(), f.Name, f.Type.Elem().Name(), ConstructorMethodName, result[0].Type().Name())
					}
				} else {
					if result[0].Type().Elem().AssignableTo(f.Type) {
						vf.Set(result[0].Elem())
					} else {
						return errors.Errorf("%s.%s: %s.%s() returns incompatible type %v", c.structType.Name(), f.Name, f.Type.Name(), ConstructorMethodName, result[0].Type().Name())
					}
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

	revision := 1
	if c.currentReleaseFunc != nil {
		revision = c.currentValue.revision + 1

		//cancel first user to release the value when all users released it
		c.currentReleaseFunc()
		c.currentReleaseFunc = nil
		c.currentValue = nil
	}

	//define new value with one user that will be released when replaced
	c.currentValue, c.currentReleaseFunc = newConfigValue(revision, structPtrValue)
	log.Debugf("Successfully loaded %s", c.structType.Name())
	return nil
} //configurable.load()

func (c *configurable) Notify(name string) {
	log.Debugf("%v: NOTIFIER(%s)", c.structType.Name(), name)
	if err := c.load([]string{name}); err != nil {
		log.Errorf("failed to load change to %s", name)
	} else {
		log.Debugf("loaded change to %s", name)
	}
}

func (c *configurable) Use() (structValue interface{}, release func()) {
	return c.currentValue.use(logger.GetCaller(1))
}

func newConfigValue(revision int, structPtrValue reflect.Value) (cv *configValue, release func()) {
	cv = &configValue{
		structPtrValue: structPtrValue,
		ts:             time.Now(),
		revision:       revision,
		users:          map[string]bool{},
		releaseChan:    make(chan string),
	}
	cv.logID = fmt.Sprintf("%s(#%d,%s)", cv.structPtrValue.Elem().Type().Name(), cv.revision, cv.ts)
	_, release = cv.use(logger.GetCaller(0))

	//start background cleanup function that will terminate when released
	go func(cv *configValue) {
		log.Debugf("%s: Created", cv.logID)
		cv.cleanup()
		log.Debugf("%s: Released", cv.logID)
	}(cv)
	return
}

type configValue struct {
	sync.Mutex
	logID          string
	structPtrValue reflect.Value
	revision       int
	ts             time.Time
	users          map[string]bool
	releaseChan    chan string
}

func (cv *configValue) use(caller logger.ICaller) (structValue interface{}, release func()) {
	cv.Lock()
	defer cv.Unlock()
	callerID := caller.String()
	cv.users[callerID] = true
	release = func() {
		cv.releaseChan <- callerID
	}
	return cv.structPtrValue.Elem().Interface(), release
}

func (cv *configValue) cleanup() {
	for len(cv.users) > 0 {
		id := <-cv.releaseChan
		delete(cv.users, id)
		log.Debugf("%s: Release, %d users remain", cv.logID, len(cv.users))
	}
	//destroy constructed fields
	log.Debugf("%s: No users remain. Destroying...", cv.logID)

	//allocate a new copy of the struct
	structType := cv.structPtrValue.Type().Elem()
	for i := 0; i < structType.NumField(); i++ {
		f := structType.Field(i)
		vf := cv.structPtrValue.Elem().Field(i)

		//check if field has a destructor
		if f.Type.Kind() == reflect.Ptr {
			//struct field stores constructed ptr
			if _, hasPtrDestructor := f.Type.MethodByName(DestructorMethodName); hasPtrDestructor {
				log.Debugf("calling (*%s)%s() ...", f.Type.Name(), DestructorMethodName)
				vf.MethodByName(DestructorMethodName).Call(nil)
			} else if _, hasValDestructor := f.Type.Elem().MethodByName(DestructorMethodName); hasValDestructor {
				log.Debugf("calling (%s)%s() ...", f.Type.Name(), DestructorMethodName)
				vf.Elem().MethodByName(DestructorMethodName).Call(nil)
			}
		} else {
			//struct field stores constructed val (not ptr)
			if _, hasPtrDestructor := f.Type.MethodByName(DestructorMethodName); hasPtrDestructor {
				log.Debugf("calling (*%s)%s() ...", f.Type.Name(), DestructorMethodName)
				vf.MethodByName(DestructorMethodName).Call(nil)
			} // else if _, hasValDestructor := f.Type.MethodByName(DestructorMethodName); hasValDestructor {
			// 	log.Debugf("calling (%s)%s() ...", f.Type.Name(), DestructorMethodName)
			// 	vf.MethodByName(DestructorMethodName).Call(nil)
			// }
		}
	} //for each field

	//destrcuted all fields
	log.Debugf("%s: Destroyed", cv.logID)
} //configValue.cleanup()

func stringInList(s string, list []string) bool {
	for _, ls := range list {
		if s == ls {
			return true
		}
	}
	return false
}

var (
	log              = logger.ForThisPackage()
	mutex            sync.Mutex
	configurableList = make([]*configurable, 0)
)

//todo:
//- add usage on configurable value
//- keep timestamp and details of error last load and success (both) and count errors - clear when load success
//- function to iterate over config and example of how to document it - build into ms
//- load arrays of structs optional/required
