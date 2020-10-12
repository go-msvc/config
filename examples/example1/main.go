package main

import (
	"net/http"
	"time"

	"github.com/go-msvc/config"
	"github.com/go-msvc/errors"
	"github.com/go-msvc/logger"
)

var log = logger.ForThisPackage()

func main() {
	logger.Top().WithStream(logger.Terminal(logger.LogLevelDebug))
	log.Debugf("Start")

	//for testing, set config values in memory
	//in real app, you will add a source to load it from db or file or env...
	config.Set("name", "Test123")
	config.Set("server", map[string]interface{}{"addr": "localhost:5555"})

	//install our config struct which will create our server
	myConfig := config.MustAdd(myConfigStruct{})
	log.Debugf("initial loaded config: %v\n", myConfig)

	//get current config struct values to show the created server
	c1 := myConfig.Current().(myConfigStruct)
	log.Debugf("server(addr=%s) = %+v", c1.Server.Addr(), c1.Server)

	//change the config to trigger new server creation with another address
	config.Set("server", map[string]interface{}{"addr": "localhost:6666"})

	//we expect a new server to exist with new config
	//and the old server still exists because we kept a reference to it in c1
	c2 := myConfig.Current().(myConfigStruct)
	log.Debugf("server(addr=%s) = %+v", c2.Server.Addr(), c2.Server)
	if c2.Server.Addr() != "localhost:6666" {
		panic(errors.Errorf("after Set() addr is wrong: %s", c2.Server.Addr()))
	}

	//todo: when do we destroy config... need use count in context
	//we should see here that server stopped since no use indicated
	//...

	//new server now listening on the other port
	//old server will run until it terminates
	time.Sleep(time.Second)
}

//config is a struct containing:
//- configurable values (public + json tag)
//- configurable things (public + json tag + type that implement config.IConfigurable)
//if config implements IValidator, it will be called each time config updated to validate
type myConfigStruct struct {
	Name   string `json:"name" doc:"Name is a configurable value"`
	Server Server `json:"server" doc:"HTTP Server"`
}

func (c myConfigStruct) Validate() error {
	if c.Name == "" {
		return errors.Errorf("missing name")
	}
	return nil
}

//server implements config.IConfigurable
type Server struct {
	cfg serverConfig
}

func (s Server) Create(c serverConfig) (Server, error) {
	log.Debugf("Creating server %v\n", c)
	s.cfg = c
	go func() {
		http.ListenAndServe(c.Addr, nil)
	}()
	log.Debugf("Created server %v\n", c)
	return s, nil
}

func (s Server) Addr() string {
	return s.cfg.Addr
}

type serverConfig struct {
	Addr string `json:"addr" default:"0.0.0.0:12345" doc:"Address"`
}
