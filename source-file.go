package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-msvc/data"
)

func File(filename string) Source {
	f, err := os.Open(filename)
	if err != nil {
		panic(fmt.Sprintf("cannot open config file %s: %+v", filename, err))
	}
	defer f.Close()
	var data map[string]interface{}
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		panic(fmt.Sprintf("cannot read JSON object from file %s: %+v", filename, err))
	}
	return file{
		data: data,
	}
} //File()

type file struct {
	data map[string]interface{}
}

func (f file) GetInto(name string, tmpl interface{}) (interface{}, error) {
	return data.GetInto(f.data, name, tmpl)
}
