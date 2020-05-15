package config

import (
	"reflect"
	"sync"

	"github.com/go-msvc/errors"
	logger "github.com/go-msvc/log"
)

var log = logger.ForThisPackage()

//Sources of config
func Sources() ISources {
	return allSources
}

//Get ...
func Get(name string, tmpl IData) (IData, error) { return allSources.Get(name, tmpl) }

//ISources is a collection of config sources
type ISources interface {
	Reset()        //removes all sources
	Add(s ISource) //add in order of execution

	Get(name string, tmpl IData) (IData, error)
	GetAll(name string) map[string]interface{}
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
	log.Info("debug all config sources")
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
func (s *sources) Get(name string, tmpl IData) (IData, error) {
	log.Debugf("sources(%p).Get(%s,%T)", s, name, tmpl)
	if tmpl == nil {
		return nil, errors.Errorf("config.Get(%s,tmpl==nil)", name)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, source := range s.list {
		config, err := source.Get(name, tmpl)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get %s", name)
		}
		if config != nil {
			if reflect.TypeOf(config) != reflect.TypeOf(tmpl) {
				panic(errors.Errorf("source %T.Get(%s) -> %T != %T", source, name, config, tmpl))
			}
			return config, nil
		}
		log.Debugf("%s not found in %s", name, source.Name())
	}
	//not found in any source, or no sources...
	return nil, nil
}

func (s *sources) GetAll(name string) map[string]interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	all := make(map[string]interface{})
	for _, source := range s.list {
		if sourceList := source.GetAll(name); len(sourceList) > 0 {
			//merge list into all
			for n, d := range sourceList {
				if _, ok := all[n]; ok {
					log.Errorf("Ignore duplicate config %s:%s", source.Name(), n)
				} else {
					all[n] = d
				}
			}
		}
	}
	log.Debugf("Found %d %s.*", len(all), name)
	for n, v := range all {
		log.Debugf("  [%s]:(%T):%v", n, v, v)
	}
	return all
} //sources.GetAll()
