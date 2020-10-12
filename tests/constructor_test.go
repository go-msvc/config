package tests_test

import (
	"fmt"
	"testing"

	"github.com/go-msvc/config"
	"github.com/go-msvc/config/source/mem"
	"github.com/go-msvc/errors"
	"github.com/go-msvc/logger"
)

var log logger.ILogger

func init() {
	logger.Top().WithStream(logger.Terminal(logger.LogLevelDebug))
	log = logger.ForThisPackage()
}

//car is constructed from config
type car struct {
	id string
}

func (car) Create(cfg carConfig) (*car, error) {
	return &car{
		id: fmt.Sprintf("%s.%d.%s", cfg.Man, cfg.Year, cfg.Mod),
	}, nil
}

type carConfig struct {
	Man  string `json:"man"`
	Mod  string `json:"mod"`
	Year int    `json:"year"`
}

func (cfg carConfig) Validate() error {
	if cfg.Man == "" || cfg.Mod == "" || cfg.Year <= 0 {
		return errors.Errorf("invalid car config %+v", cfg)
	}
	return nil
}

func TestConCar(t *testing.T) {
	config.Sources().Reset()
	cfgInMemory := mem.New().With("bakkie", map[string]interface{}{"man": "Toyota", "mod": "Hilux", "year": 2009})
	config.Sources().Add(cfgInMemory)

	type configWithBakkie struct {
		Bakkie car `json:"bakkie"`
	}
	c := config.MustAdd(configWithBakkie{})
	p, rel := c.Use()
	defer rel()
	pp := p.(configWithBakkie)
	if pp.Bakkie.id != "Toyota.2009.Hilux" {
		t.Fatalf("wrong values: %+v", p)
	}
}

func TestConMissingCar(t *testing.T) {
	config.Sources().Reset()
	cfgInMemory := mem.New()
	config.Sources().Add(cfgInMemory)

	type configWithBakkie struct {
		Bakkie car `json:"bakkie"`
	}
	_, err := config.Add(configWithBakkie{})
	if err == nil {
		t.Fatalf("added without error while config is missing")
	}
}

func TestConOptDefined(t *testing.T) {
	config.Sources().Reset()
	cfgInMemory := mem.New().With("bakkie", map[string]interface{}{"man": "Toyota", "mod": "Hilux", "year": 2009})
	config.Sources().Add(cfgInMemory)

	type configWithBakkie struct {
		Bakkie *car `json:"bakkie"` //optional because ptr
	}
	c := config.MustAdd(configWithBakkie{})
	p, rel := c.Use()
	defer rel()
	pp := p.(configWithBakkie)
	if pp.Bakkie == nil || pp.Bakkie.id != "Toyota.2009.Hilux" {
		t.Fatalf("wrong values: %+v", pp)
	}
}

func TestConOptAbsent(t *testing.T) {
	config.Sources().Reset()
	cfgInMemory := mem.New()
	config.Sources().Add(cfgInMemory)

	type configWithBakkie struct {
		Bakkie *car `json:"bakkie"` //optional because ptr
	}
	c := config.MustAdd(configWithBakkie{})
	p, rel := c.Use()
	defer rel()
	pp := p.(configWithBakkie)
	if pp.Bakkie != nil {
		t.Fatalf("loaded while absent")
	}
}

func TestConRepeated(t *testing.T) {
	config.Sources().Reset()
	cfgInMemory := mem.New().
		With("hilux", map[string]interface{}{"man": "Toyota", "mod": "Hilux", "year": 2009}).
		With("ford", map[string]interface{}{"man": "Ford", "mod": "Ranger", "year": 2011})
	config.Sources().Add(cfgInMemory)

	type configWithBakkie struct {
		Hilux car `json:"hilux"`
		Ford  car `json:"ford"`
	}
	c := config.MustAdd(configWithBakkie{})
	p, rel := c.Use()
	defer rel()
	pp := p.(configWithBakkie)
	if pp.Hilux.id != "Toyota.2009.Hilux" || pp.Ford.id != "Ford.2011.Ranger" {
		t.Fatalf("loaded invalid values %+v", pp)
	}
}

func TestConRepeatedOptional(t *testing.T) {
	config.Sources().Reset()
	cfgInMemory := mem.New().
		//With("hilux", map[string]interface{}{"man": "Toyota", "mod": "Hilux", "year": 2009}).
		With("ford", map[string]interface{}{"man": "Ford", "mod": "Ranger", "year": 2011})
	config.Sources().Add(cfgInMemory)

	type configWithBakkie struct {
		Hilux *car `json:"hilux"`
		Ford  *car `json:"ford"`
	}
	c := config.MustAdd(configWithBakkie{})
	p, rel := c.Use()
	defer rel()
	pp := p.(configWithBakkie)
	if pp.Hilux != nil || pp.Ford == nil || pp.Ford.id != "Ford.2011.Ranger" {
		t.Fatalf("loaded invalid values %+v", pp)
	}
}
