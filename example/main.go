package main

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-msvc/config"
	"github.com/go-msvc/data"
	"github.com/go-msvc/errors"
	"github.com/go-msvc/logger"
)

var log = logger.New().WithLevel(logger.LevelDebug)

type Logger interface {
	Log(m string)
}

type LoggerPrefixConfig struct {
	Prefix string `json:"prefix"`
}

func (c LoggerPrefixConfig) Create() (Logger, error) {
	return loggerWithPrefix{config: c}, nil
}

type loggerWithPrefix struct {
	config LoggerPrefixConfig
}

func (l loggerWithPrefix) Log(m string) {
	fmt.Printf("%s %s\n", l.config.Prefix, m)
}

type LoggerSuffixConfig struct {
	Suffix string `json:"suffix"`
}

func (c LoggerSuffixConfig) Create() (Logger, error) {
	return loggerWithSuffix{config: c}, nil
}

type loggerWithSuffix struct {
	config LoggerSuffixConfig
}

func (l loggerWithSuffix) Log(m string) {
	fmt.Printf("%s %s\n", m, l.config.Suffix)
}

type HttpServerConfig struct {
	Addr string
}

func (c HttpServerConfig) Create() (Server, error) {
	return httpServer{config: c}, nil
}

type httpServer struct {
	config HttpServerConfig
}

func (s httpServer) Serve(ms MicroService) error {
	return errors.Errorf("NYI")
}

func init() {
	//the implementation package must register its implementation
	//there could be many different implementations
	config.RegisterConstructor("prefix", LoggerPrefixConfig{})
	config.RegisterConstructor("suffix", LoggerSuffixConfig{})

	config.RegisterConstructor("http", HttpServerConfig{})

	//the package that will use the constructed item must tell config
	//the required item, not specifying which implementation to use
	//just the interface that must be constructed from this config item
	config.MustConstruct("ms.server", reflect.TypeOf((*Server)(nil)).Elem())
}

func main() {
	//load the config - all the required config and construct items as indicated
	//construction will happen in this call
	if err := config.Load(); err != nil {
		panic(fmt.Sprintf("config error: %+v", err))
	}

	//now the config data and constructed items can be fetched and used
	//as many times as you need (returning the same values each time)
	server, ok := config.Get("ms.server").(Server)
	if !ok {
		panic(fmt.Sprintf("failed to create configured ms.server"))
	}

	if err := server.Serve(nil); err != nil {
		panic(fmt.Sprintf("server failed: %+v", err))
	}
} //main()

// this program needs a server to serve the micro-service
type Server interface {
	Serve(ms MicroService) error
}

// the micro-service is not important in this config example...
type MicroService interface {
	//...
}

// to get the server config we expect something like this
// {"ms":{"server":{...}}}
// inside that must be one of the registered types of server implementations, e.g.
//
//	"http":{"addr":"localhost:12345"}
//
// or
//
//	"nats":{"url":"nats//localhost:4222", "consume":"some-qname"}
//
// so we need to tell config to load a server constructor from "ms.server"

// To get the interface type is a bit technical...
// see https://stackoverflow.com/questions/7132848/how-to-get-the-reflect-type-of-an-interface
// e.g. t := reflect.TypeOf((*error)(nil)).Elem()
func init() {
	interfaceType := reflect.TypeOf((*Server)(nil)).Elem()
	config.MustConstruct("ms.server", interfaceType)
	//after that, config knows we need some ServerConfig under ms.server
	//which must be an object with the name of the implementation and its value
	// "<implementation name>":{...}
}

// register a custom source of config to use hardcoded values
// ... just to demonstrate how you can create more config sources!
func init() {
	config.AddSource("custom", hardCodedConfig{})
}

type hardCodedConfig struct {
}

func (hardCodedConfig) GetInto(ref string, tmpl interface{}) (interface{}, error) {
	log.Debugf("hard.Get(%s)", ref)
	jsonValue := []byte(`
	{
		"ms":{
			"server":{
				"http":{
					"addr":"localhost:55555"
				}
			}
		}
	}`)

	var configData interface{}
	if err := json.Unmarshal(jsonValue, &configData); err != nil {
		return nil, errors.Wrapf(err, "invalid JSON")
	}
	return data.Get(configData, ref)
}

// if configSource := os.Getenv("CONFIG_SOURCE"); configSource != "" {
// 	//add source from env var, expecting value like <type>:<value>
// 	//e.g. export CONFIG_SOURCE="file://config.json"
// 	//or   export CONFIG_SOURCE="etcd://localhost:11111"
// 	//or   export CONFIG_SOURCE="http://localhost:12345/config/test" (will fetch with GET)
// }
