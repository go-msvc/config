package mem

import (
	"reflect"
	"sync"

	"github.com/go-msvc/config"
	"github.com/go-msvc/errors"
	"github.com/go-msvc/logger"
)

type IMemSource interface {
	config.ISource
	Set(name string, value interface{}) error
}

var log = logger.ForThisPackage()

type memSource struct {
	sync.Mutex
	byName    map[string]interface{}
	notifiers map[string][]config.INotifier
}

//New ...
func New() IMemSource {
	return &memSource{
		byName:    map[string]interface{}{},
		notifiers: make(map[string][]config.INotifier),
	}
}

func (s *memSource) Name() string {
	return "inMemory"
}

func (s *memSource) Get(name string, notifier config.INotifier) (interface{}, error) {
	s.Lock()
	defer s.Unlock()
	if v, ok := s.byName[name]; ok {
		if notifier != nil {
			if s.notifiers[name] == nil {
				s.notifiers[name] = []config.INotifier{notifier}
			} else {
				s.notifiers[name] = append(s.notifiers[name], notifier)
			}
		}
		return v, nil
	}
	return nil, nil
}

func (s *memSource) Set(name string, value interface{}) error {
	notifiers, err := s.set(name, value)
	if err != nil {
		return errors.Wrapf(err, "failed to set")
	}
	if notifiers != nil {
		for _, n := range notifiers {
			n.Notify(name)
		}
	}
	return nil
}

func (s *memSource) set(name string, value interface{}) ([]config.INotifier, error) {
	s.Lock()
	defer s.Unlock()
	if value == nil {
		//delete only if exists
		if _, ok := s.byName[name]; !ok {
			return nil, nil //did not exist
		}
		delete(s.byName, name)
	} else {
		//set only if different or new
		if existingValue, ok := s.byName[name]; !ok {
			s.byName[name] = value
		} else if !reflect.DeepEqual(existingValue, value) {
			s.byName[name] = value
		} else {
			return nil, nil //no change
		}
	}

	//changed/new
	//get notifiers and reset them for the new value
	//the caller must call the notifiers after this function released the lock
	notifiers, ok := s.notifiers[name]
	if ok {
		s.notifiers[name] = nil //do not call same notifiers ever again
	}
	return notifiers, nil //notifiers are returned and lock released before they are called, which is likely to call Get() again that needs the lock
}
