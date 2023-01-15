package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/go-msvc/data"
	"github.com/go-msvc/errors"
	"github.com/go-msvc/logger"
)

// indicate that your module requires a configurable value
// call this before config.Load(), i.e. in your package's init() func
// because config is loaded at the start of main()
// tmpl is your default config struct, and may be empty or have some
// default content,
// it should preferably also implement Validator interface
// not matter if called multiple times, as long it has the same tmpl
// MayConfigure for optional config - Get() will return nil if not configured
// Must for required config - Get() will return value
func MayConfigure(ref string, tmpl interface{}) {
	flagConfigure(ref, tmpl, false)
	//todo: optional not yet implemented... will fail if not configured
}

func MustConfigure(ref string, tmpl interface{}) {
	flagConfigure(ref, tmpl, true)
}

func MayConstruct(ref string, constructedType reflect.Type) {
	flagConstruct(ref, constructedType, false)
	//todo: optional not yet implemented... will fail if not configured
}

func MustConstruct(ref string, constructedType reflect.Type) {
	flagConstruct(ref, constructedType, true)
}

func flagConfigure(ref string, tmpl interface{}, required bool) {
	if loaded {
		panic(fmt.Sprintf("config.MustConfigure(%s) called after config.Load()", ref))
	}
	if !validReference(ref) {
		panic(fmt.Sprintf("invalid config reference(%s), expecting dot-notation reference", ref))
	}
	if tmpl == nil {
		panic(fmt.Sprintf("MustConfigure(%s) cannot configure nil", ref))
	}
	if existingTmpl, ok := mustConfigureByRef[ref]; ok {
		if reflect.TypeOf(tmpl) != reflect.TypeOf(existingTmpl) {
			panic(fmt.Sprintf("config.MustConfigure(%s) with conflicting type %v != %v already required", ref, reflect.TypeOf(tmpl), reflect.TypeOf(existingTmpl)))
		}
	} else {
		mustConfigureByRef[ref] = tmpl
	}
} //flagConfigure()

func flagConstruct(ref string, constructedType reflect.Type, required bool) {
	if loaded {
		panic(fmt.Sprintf("config.MustConstruct(%s) called after config.Load()", ref))
	}
	if !validReference(ref) {
		panic(fmt.Sprintf("invalid config reference(%s), expecting dot-notation reference", ref))
	}
	if constructedType == nil {
		panic(fmt.Sprintf("MustConstruct(%s) cannot construct nil", ref))
	}
	info := constructorInfoFor(constructedType)
	info.Lock()
	defer info.Unlock()
	info.mustConstructByRef[ref] = true
	log.Infof("MUST Construct \"%s\"", ref)
} //flagConstruct()

// Register a constructor implementation
// tmpl is the config with optional default values and implements Validator interface
func RegisterConstructor(name string, tmpl interface{}) {
	if loaded {
		panic(fmt.Sprintf("config.RegisterConstructor(%s) called after config.Load()", name))
	}

	//tmpl must have a method Create() that returns some interface type or error
	tmplType := reflect.TypeOf(tmpl)
	createMethod, ok := tmplType.MethodByName("Create")
	if !ok {
		panic(fmt.Sprintf("constructor type %T has no method called Create()", tmpl))
	}
	if createMethod.Type.NumIn() > 1 { //expect 1 because its an object method (like passing self in python)
		panic(fmt.Sprintf("%T.Create(...) may not take any arguments", tmpl))
	}
	if createMethod.Type.NumOut() != 2 {
		panic(fmt.Sprintf("%T.Create(...) must return (<YourInterfaceType>,error)", tmpl))
	}
	if !createMethod.Type.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		panic(fmt.Sprintf("%T.Create(...) must return (<YourInterfaceType>,error)", tmpl))
	}

	constructedType := createMethod.Type.Out(0)
	if createMethod.Type.Out(0).Kind() != reflect.Interface {
		panic(fmt.Sprintf("%T.Create(...) must return (<YourInterfaceType>,error)", tmpl))
	}

	info := constructorInfoFor(constructedType)
	if _, ok := info.tmplByName[name]; ok {
		panic(fmt.Sprintf("%v constructor(name=\"%s\") is already registered!", constructedType, name))
	}
	info.tmplByName[name] = tmpl
	log.Debugf("Registered %s constructor(name=\"%s\"): %T", constructedType, name, tmpl)
} //RegisterConstructor()

// load config from all config sources
// after this, sources cannot be added and Required will also fail
// to ensure that documentation can be generated and config
// is not used on some code branch that was not known at the start
// this process will load and construct all the items marked with
// calls to Required() and MustConstruct()
func Load() error {
	moduleDataMutex.Lock()
	defer moduleDataMutex.Unlock()

	if loaded {
		return nil //already loaded
	}

	if len(sources) == 0 {
		return errors.Errorf("no sources of config were added (call config.AddSource(...))")
	}

	//get all MustConfigure() values from the available sources
	//the first value is used, so multiple sources can be specified for redundancy
	//or to support a mix of sources
	configByRef = map[string]interface{}{}
	for ref, requiredTmpl := range mustConfigureByRef {
		found := false
		for _, ns := range sources {
			configuredValue, err := ns.source.GetInto(ref, requiredTmpl)
			if err != nil {
				//expect value and err nil if not configured in this source,
				//so this is treated as an error in the source, e.g.
				//cannot connect to source or authentication error or
				//invalid values
				return errors.Wrapf(err, "failed to get source(%s).config(%s)", ns.name, ref)
			}
			if configuredValue != nil {
				configByRef[ref] = configuredValue
				log.Debugf("Source(%s).Configured(%s): %T", ns.name, ref, configuredValue)
				found = true
				break //skip other sources
			}
		}
		if !found {
			return errors.Errorf("config(%s) not found in any source", ref)
		}
	} //for each required config

	//construct all required items
	//start first by getting all the required values from sources
	//so we can fail on missing/invalid config before any construction code is called
	constructorByRef := map[string]interface{}{}
	for constructedType, info := range constructorsByType {
		for ref := range info.mustConstructByRef {
			if len(info.tmplByName) == 0 {
				return errors.Errorf("config(%s) cannot load without any registered constructors for %v", ref, constructedType)
			}
			found := false
			var implNamedConfig map[string]interface{}
			var ns namedSource
			for _, ns = range sources {
				value, err := ns.source.GetInto(ref, map[string]interface{}{})
				if err != nil {
					//expect value and err nil if not configured in this source,
					//so this is treated as an error in the source, e.g.
					//cannot connect to source or authentication error or
					//invalid values
					return errors.Wrapf(err, "failed to get source(%s).config(%s)", ns.name, ref)
				}
				if value != nil {
					//store the value for processing below
					implNamedConfig = value.(map[string]interface{})
					log.Debugf("Source(%s).Configured(%s)", ns.name, ref)
					found = true
					break //skip other sources
				}
			}
			if !found {
				return errors.Errorf("config(%s) not found in any source", ref)
			}

			if len(implNamedConfig) == 0 {
				return errors.Errorf("source(%s).config(%s) does identify an implementation as {\"<impl>\":{...}}", ns.name, ref)
			}
			if len(implNamedConfig) > 1 {
				return errors.Errorf("source(%s).config(%s) identifies multiple implementations {\"<impl>\":{...}, ...} instead of just one", ns.name, ref)
			}
			var implName string
			for implName = range implNamedConfig {
				//do nothing
			}

			//get the named implementation (must have been registered with RegisterConstructor(<implName>, ...))
			constructorTmpl, ok := info.tmplByName[implName]
			if !ok {
				registeredNames := []string{}
				for n := range info.tmplByName {
					registeredNames = append(registeredNames, n)
				}
				//if you get this error, config.RegisterConstructor(<implName>, ...) was not called
				//or you misspelled the <implName> in config <ref>:{<implName>:{...}}
				return errors.Errorf("config(%s) has no constructor for \"%s\", only for %s", ref, implName, strings.Join(registeredNames, "|"))
			}

			// get the config value into the constructor tmpl by calling the source again
			// this time including the implName and the tmpl
			constructorRef := ref + "." + implName
			constructorValue, err := ns.source.GetInto(constructorRef, constructorTmpl)
			if err != nil {
				return errors.Wrapf(err, "failed to get source(%s).config(%s)", ns.name, constructorRef)
			}
			if constructorValue == nil {
				//likely internal software error - should not get here after above checks - just checked for sanity
				return errors.Wrapf(err, "failed to get source(%s).config(%s)", ns.name, constructorRef)
			}

			//this is valid - proceed to next MustConstruct(ref) then construction will be
			//done below...
			if _, ok := reflect.TypeOf(constructorValue).MethodByName("Create"); !ok {
				//seems source did not return constructorTmpl as it should!
				//try to fix it
				if converted, err := data.GetInto(constructorValue, "", constructorTmpl); err == nil {
					constructorByRef[ref] = converted
					log.Debugf("source(%s).Get(%s) -> %T != %T but fixed, now %T", ns.name, constructorRef, constructorValue, constructorTmpl, converted)
				} else {
					return errors.Errorf("source(%s).Get(%s) -> %T != %T and cannot fix it... check your config source", ns.name, constructorRef, constructorValue, constructorTmpl)
				}
			} else {
				log.Debugf("%s: %T:%+v", constructorRef, constructorValue, constructorValue)
				constructorByRef[ref] = constructorValue
			}
		}
	}

	//all config read and validated, now do all the constructions
	for constructorRef, configured := range constructorByRef {
		//call Create() method
		method := reflect.ValueOf(configured).MethodByName("Create")
		results := method.Call(nil)
		if !results[1].IsNil() {
			return errors.Wrapf(results[0].Interface().(error), "failed to construct %s", constructorRef)
		}
		if results[0].IsNil() {
			return errors.Errorf("%T.Create() returned nil,nil", configured)
		}

		//store without implName (e.g. "ms.server" and not "ms.server.http")
		created := results[0].Interface()
		configByRef[constructorRef] = created
		log.Debugf("Constructed(%s): %T", constructorRef, created)
	}

	loaded = true
	return nil
} //Load()

// Get an item that you specified with MustConfigure() or MustConstruct()
// by the time you call this, the config must exist
// and this call will panic if not
func Get(ref string) any {
	if !loaded {
		panic("config.Load() not yet done")
	}
	if v, ok := configByRef[ref]; ok {
		return v
	}
	panic(fmt.Sprintf("config(%s) not found. Make sure you called config.MustConfigure() or config.MustConstruct()", ref))
} //Get()

var log = logger.New().WithLevel(logger.LevelDebug)

type Constructor interface {
	Create() (interface{}, error)
}

// names may only have [a-zA-Z0-9_-] characters, start with a letter and end with letter or digit
const namePattern = `[a-zA-Z]([a-zA-Z0-9_-]*[a-zA-Z0-9])*`

var nameRegex = regexp.MustCompile("^" + namePattern + "$")

func validName(name string) bool {
	return nameRegex.MatchString(name)
}

// reference currently only support dot-notation of valid names
// in future, consider it may also support array index like jq if multiple items are required
// but for now, the user should manage lists inside a parent struct type
const refPattern = namePattern + `(\.` + namePattern + `)*`

var refRegex = regexp.MustCompile("^" + refPattern + "$")

func validReference(ref string) bool {
	return refRegex.MatchString(ref)
}

var (
	moduleDataMutex    sync.Mutex
	mustConfigureByRef = map[string]interface{}{}
	constructorsByType = map[reflect.Type]*constructorInfo{}
	loaded             = false
	configByRef        = map[string]interface{}{} //loaded or constructed values from Load()
)

type constructorInfo struct {
	sync.Mutex

	//this is the config structures by implementation name
	//e.g. tmplByName["http"] = HttpServerConfig{}
	//it's called "tmpl" because it is the template of config that we use
	//when loading the config into a new copy of the specified structure
	tmplByName map[string]interface{} //registered implementations, value is config tmpl

	//these are the constructed items by config name
	//key is the config key, excluding the implementation name
	//e.g. if you configure ms.server.http:{...}
	//then http identifies the implementation of the ms.server
	//and the key will be: constructed["ms.server"] = ...
	//the value is the constructed item, i.e. the Server in this case
	//which will be an HTTP version of Server.
	//the values are defined in MustConstruct(name, type) as:
	//		constructorByType[type].mustConstructedByName[name] = true
	//then during config.Load(), the constructors are called and
	//the constructed value is
	//		constructorByType[type].constructedByName[name] = ...
	mustConstructByRef map[string]bool
	constructedByName  map[string]interface{}
}

func constructorInfoFor(constructedType reflect.Type) *constructorInfo {
	moduleDataMutex.Lock()
	defer moduleDataMutex.Unlock()
	info, ok := constructorsByType[constructedType]
	if !ok {
		info = &constructorInfo{
			tmplByName:         map[string]interface{}{},
			mustConstructByRef: map[string]bool{},
			constructedByName:  map[string]interface{}{},
		}
		constructorsByType[constructedType] = info
	}
	return info
} //constructorInfoFor()
