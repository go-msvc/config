package config

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/go-msvc/errors"
)

//Create a configured item
func Create(path string, constructed interface{}) error {
	t := reflect.TypeOf(constructed)
	if t.Kind() != reflect.Ptr {
		return fmt.Errorf("Create(%s,%T) required pointer argument", path, constructed)
	}
	t = t.Elem()
	cl, ok := constructors[t]
	if !ok {
		return fmt.Errorf("no constructors registered to create %v %s", t, path)
	}
	log.Debugf("%d constructors for %v:", len(cl), t)

	//now look for each constructor by name to parse its own type
	//any path/<name of registered constructor(s)>, and expect only one!
	for constructorName, constructor := range cl {
		cfg, err := allSources.Get(path+"/"+constructorName, constructor)
		if err != nil {
			return errors.Wrapf(err, "failed to get config for %s", path)
		}
		if cfg == nil {
			log.Debugf("%s/%s is not configured", path, constructorName)
			continue
		}
		log.Debugf("Got %T %s: %+v", cfg, path+"/"+constructorName, cfg)

		//call the create method:
		m := reflect.ValueOf(cfg).MethodByName("Create")
		results := m.Call([]reflect.Value{})
		if results[1].Interface() != nil {
			err = results[1].Interface().(error)
			return errors.Wrapf(err, "failed to construct")
		}
		reflect.ValueOf(constructed).Elem().Set(results[0])
		log.Debugf("constructed %v %s/%s", t, path, constructorName)
		return nil
	}
	s := ""
	for n := range cl {
		s += "|" + n
	}
	return errors.Errorf("missing config for %s/[%s]", path, s[1:])
}

//RegisterConstructor ...
func RegisterConstructor(n string, c IConstructor) {
	if n == "" || c == nil {
		panic(fmt.Errorf("RegisterConstructor(%s,%p)", n, c))
	}
	//c must be a struct
	t := reflect.TypeOf(c)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic(fmt.Errorf("RegisterConstructor(%T) expects a struct implementing %T", c, ((*IConstructor)(nil))))
	}
	//t must implement Create() method returning (IConstructed,error)
	if _, ok := t.MethodByName("Create"); !ok {
		panic(fmt.Errorf("RegisterConstructor(%T) has no method Create()(IConstructed,error)", c))
	}
	m := reflect.ValueOf(c).MethodByName("Create")
	log.Debugf("%T.Create() has %d args and return %d values", c, m.Type().NumIn(), m.Type().NumOut())
	if m.Type().NumIn() != 0 {
		panic(fmt.Errorf("RegisterConstructor: %T.Create() should take no arguments", c))
	}
	if m.Type().NumOut() != 2 {
		panic(fmt.Errorf("RegisterConstructor: %T.Create() should return 2 results: (<IConstructed type>,error)", c))
	}
	log.Debugf("%T.Create() returns (%v,%v)", c, m.Type().Out(0), m.Type().Out(1))

	resultType := m.Type().Out(0)
	constructedInterface := reflect.TypeOf((*IConstructed)(nil)).Elem()
	if !resultType.Implements(constructedInterface) {
		panic(fmt.Errorf("%T.Create() 1st result %v does not implement %v", c, resultType, constructedInterface))
	}

	errorInterface := reflect.TypeOf((*error)(nil)).Elem()
	if !m.Type().Out(1).Implements(errorInterface) {
		panic(fmt.Errorf("%T.Create() 2nd result %v does not implement error", c, m.Type().Out(1)))
	}

	//get list of constructors with this result:
	constructorMutex.Lock()
	defer constructorMutex.Unlock()
	cl, ok := constructors[resultType]
	if !ok {
		cl = map[string]IConstructor{}
		constructors[resultType] = cl
	}
	if exist, ok := cl[n]; ok {
		panic(fmt.Errorf("registering %T duplicate [%T][%s]=%T", c, resultType, n, exist))
	}
	cl[n] = c
	log.Infof("Registered %s constructor(%s): %T", resultType, n, c)
}

var (
	constructorMutex sync.Mutex
	constructors     = map[reflect.Type]map[string]IConstructor{}
)

//IConstructor ...
type IConstructor interface {
	IValidator
	//checked at run-time to have: Create() (<type>,error)
}

//IConstructed ...
type IConstructed interface {
}

//IValidator ...
type IValidator interface {
	Validate() error
}
