package config

import (
	"sync"

	"github.com/jansemmelink/log"
)

//ISource ...
type ISource interface {
	Name() string
	Get(name string, tmpl IData) (IData, error)
	GetAll(name string) map[string]interface{}
}

//ISources is a collection of config sources
type ISources interface {
	Add(s ISource)
	With(s ISource) ISources
	Get(name string, tmpl IData) (IData, error)
	GetAll(name string) map[string]interface{}
}

//NewSources makes a new collection of config sources
func NewSources() ISources {
	return &sources{
		list: make([]ISource, 0),
	}
}

type sources struct {
	mutex sync.Mutex
	list  []ISource
}

//Add to default list of config sources used by Get(name)
func (sources *sources) Add(s ISource) {
	if s == nil {
		return
	}
	sources.mutex.Lock()
	defer sources.mutex.Unlock()
	for _, existing := range sources.list {
		if s == existing {
			return
		}
	}
	sources.list = append(sources.list, s)
} //sources.Add()

func (sources *sources) With(s ISource) ISources {
	if s == nil {
		return sources
	}
	sources.mutex.Lock()
	defer sources.mutex.Unlock()
	for _, existing := range sources.list {
		if s == existing {
			return sources
		}
	}
	sources.list = append(sources.list, s)
	return sources
} //sources.With()

func (sources *sources) Get(name string, tmpl IData) (IData, error) {
	sources.mutex.Lock()
	defer sources.mutex.Unlock()
	for _, source := range sources.list {
		config, err := source.Get(name, tmpl)
		if err != nil {
			return nil, log.Wrapf(err, "error in %s", name)
		}
		if config != nil {
			return config, nil
		}
		log.Debugf("%s not found in %s", name, source.Name())
	}
	return nil, nil
}

func (sources *sources) GetAll(name string) map[string]interface{} {
	sources.mutex.Lock()
	defer sources.mutex.Unlock()

	all := make(map[string]interface{})
	for _, source := range sources.list {
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
	log.Debugf("Found %d configs named(%s)", len(all), name)
	return all
}
