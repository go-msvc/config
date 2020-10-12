package tests_test

import (
	"testing"

	"github.com/go-msvc/config"
	"github.com/go-msvc/config/source/mem"
)

func TestPerson(t *testing.T) {
	config.Sources().Reset()
	cfgInMemory := mem.New().With("name", "Jan").With("age", 10)
	config.Sources().Add(cfgInMemory)

	personConfig := config.MustAdd(person{})
	p := personConfig.Current().(person)
	if p.Name != "Jan" || p.Age != 10 {
		t.Fatalf("wrong values: %+v", p)
	}
}

type person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestConfigWithPerson(t *testing.T) {
	config.Sources().Reset()
	cfgInMemory := mem.New().With("p1", map[string]interface{}{"name": "Jan", "age": 10})
	config.Sources().Add(cfgInMemory)

	type configWithP1 struct {
		P1 person `json:"p1"`
	}
	c := config.MustAdd(configWithP1{})
	p := c.Current().(configWithP1)
	if p.P1.Name != "Jan" || p.P1.Age != 10 {
		t.Fatalf("wrong values: %+v", p)
	}
}

func TestMissingPerson(t *testing.T) {
	config.Sources().Reset()
	cfgInMemory := mem.New() //.With("p1", map[string]interface{}{"name": "Jan", "age": 10})
	config.Sources().Add(cfgInMemory)

	type configWithP1 struct {
		P1 person `json:"p1"`
	}
	_, err := config.Add(configWithP1{})
	if err == nil {
		t.Fatalf("added without error while config is missing")
	}
}

func TestConfigWithOptionalPersonDefined(t *testing.T) {
	config.Sources().Reset()
	cfgInMemory := mem.New().With("p1", map[string]interface{}{"name": "Jan", "age": 10})
	config.Sources().Add(cfgInMemory)

	type configWithOptP1 struct {
		P1 *person `json:"p1"`
	}
	c := config.MustAdd(configWithOptP1{})
	p := c.Current().(configWithOptP1)
	if p.P1 == nil || p.P1.Name != "Jan" || p.P1.Age != 10 {
		t.Fatalf("loaded wrong values: %+v", p)
	}
}

func TestConfigWithOptionalPersonAbsent(t *testing.T) {
	config.Sources().Reset()
	cfgInMemory := mem.New() //.With("p1", map[string]interface{}{"name": "Jan", "age": 10})
	config.Sources().Add(cfgInMemory)

	type configWithOptP1 struct {
		P1 *person `json:"p1"`
	}
	c := config.MustAdd(configWithOptP1{})
	p := c.Current().(configWithOptP1)
	if p.P1 != nil {
		t.Fatalf("loaded absent p1")
	}
}

func TestConfigWithRepeatedStructAbsent(t *testing.T) {
	config.Sources().Reset()
	cfgInMemory := mem.New() //.With("p1", map[string]interface{}{"name": "Jan", "age": 10})
	config.Sources().Add(cfgInMemory)

	type configWithOptP1 struct {
		P1 *person `json:"p1"`
		P2 *person `json:"p2"`
	}
	c := config.MustAdd(configWithOptP1{})
	p := c.Current().(configWithOptP1)
	if p.P1 != nil || p.P2 != nil {
		t.Fatalf("loaded absent p1=%p or p2=%p", p.P1, p.P2)
	}
}

func TestConfigWithRepeatedStruct(t *testing.T) {
	config.Sources().Reset()
	cfgInMemory := mem.New().
		With("p1", map[string]interface{}{"name": "Jan", "age": 10}).
		With("p2", map[string]interface{}{"name": "Koos", "age": 20})
	config.Sources().Add(cfgInMemory)

	type configWithOptP1 struct {
		P1 *person `json:"p1"`
		P2 *person `json:"p2"`
	}
	c := config.MustAdd(configWithOptP1{})
	p := c.Current().(configWithOptP1)
	if p.P1 == nil || p.P1.Name != "Jan" || p.P1.Age != 10 ||
		p.P2 == nil || p.P2.Name != "Koos" || p.P2.Age != 20 {
		t.Fatalf("loaded absent p1=%p or p2=%p", p.P1, p.P2)
	}
}
