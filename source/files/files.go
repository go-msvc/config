package files

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/go-msvc/config"
	"github.com/jansemmelink/log"
)

//New creates a source of config from files in a directory
func New(dir string) config.ISource {
	return &source{
		dir:   dir,
		files: make(map[string]*file),
	}
}

type source struct {
	dir   string
	mutex sync.Mutex
	files map[string]*file
}

func (files *source) Name() string {
	return "files(" + files.dir + ")"
}

func (files *source) file(name string) *file {
	f, err := files.getFile(name)
	if err != nil {
		log.Errorf("cannot get file: %v", err)
	}
	return f
}

//return file if found, else nil
//return error only if file exists but cannot be used e.g. invalid JSON contents
func (files *source) getFile(name string) (*file, error) {
	files.mutex.Lock()
	defer files.mutex.Unlock()

	//use existing
	if namedFilePtr, ok := files.files[name]; ok {
		return namedFilePtr, nil
	}

	//not yet created - see if exists in directory
	path := files.dir + "/" + name + ".json"
	if fileInfo, err := os.Stat(path); err == nil {
		if fileInfo.Mode().IsRegular() {
			if jsonFile, err := os.Open(path); err == nil {
				//load JSON file contents into memory
				var jsonData map[string]interface{}
				if err := json.NewDecoder(jsonFile).Decode(&jsonData); err != nil && err != io.EOF {
					return &file{
						name: "invalidJSON<\"" + name + "\">",
						path: path,
					}, log.Wrapf(err, "file(%s) has invalid JSON contents", path)
				}
				newFilePtr := &file{
					name: name,
					path: path,
					json: jsonData,
				}
				files.files[name] = newFilePtr
				log.Debugf("Loaded(%s): %+v", newFilePtr.name, newFilePtr.json)
				return newFilePtr, nil
			}
		}
	}
	return &file{
		name: "notFound<\"" + name + "\">",
		path: path,
	}, nil
} //source.getFile()

func (files *source) Get(name string, tmpl config.IData) (config.IData, error) {
	//skip leading '.'
	for len(name) > 0 && name[0] == '.' {
		name = name[1:]
	}

	//split name on first '.'
	parts := strings.SplitN(name, ".", 2)
	switch len(parts) {
	case 1:
		return files.file(parts[0]).Get("", tmpl)
	case 2:
		return files.file(parts[0]).Get(parts[1], tmpl)
	default:
	} //switch(nr parts in name)
	panic(fmt.Sprintf("Get(\"%s\") -> %d parts", name, len(parts)))
} //source.Get()

//name e.g. "abc.server" will look for file "abc"
//then object "server" then return each item in that object,
//e.g. map will consist of two items "nats" + "rest"
func (files *source) GetAll(name string) map[string]interface{} {
	all := make(map[string]interface{})
	//first part of name must be filename, e.g. "abc.server.*"
	//then expect file abc to exist with server.* inside...
	names := strings.SplitN(name, ".", 2)
	if len(names) != 2 || len(names[0]) == 0 {
		return all
	}
	f := files.file(names[0])
	if f == nil {
		return all
	}

	//got file e.g. "abc", now need to look deeper for e.g. "server.*"
	//meaning we look for server, then iterate over items inside e.g. nats + rest
	if len(names) > 1 && len(names[1]) > 0 {
		data, err := f.Get(names[1], jsonObject{})
		if err != nil {
			log.Errorf("failed to get file(%s).data(%s)", names[0], names[1])
		}
		objPtr := data.(*jsonObject)

		//add each named item from the object
		for n, v := range *objPtr {
			all[n] = v
		}
	} else {
		log.Errorf("names[1] not defined... TODO")
	}
	return all
}

type file struct {
	name string
	path string
	json map[string]interface{}
}

func (f file) Get(name string, tmpl config.IData) (config.IData, error) {
	if f.json == nil {
		return nil, nil
	}

	dataObj := jsonGet(f.json, name)
	if dataObj == nil {
		return nil, nil
	}
	jsonObj, _ := json.Marshal(dataObj)

	//got the data - parse into struct
	dataValue := reflect.New(reflect.TypeOf(tmpl))
	configData := dataValue.Interface().(config.IData)
	if err := json.Unmarshal(jsonObj, configData); err != nil {
		return nil, log.Wrapf(err, "file(%s): cannot decode %s into %T", f.name, name, configData)
	}
	if err := configData.Validate(); err != nil {
		return nil, log.Wrapf(err, "file(%s): invalid %s", f.name, name)
	}
	return configData, nil
}

type jsonObject map[string]interface{}

func (obj jsonObject) Validate() error { return nil }

func jsonGet(obj map[string]interface{}, name string) interface{} {
	//skip leading '.'
	for len(name) > 0 && name[0] == '.' {
		name = name[1:]
	}
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 0 || len(parts[0]) == 0 {
		return nil
	}

	namedValue, ok := obj[parts[0]]
	if !ok {
		return nil
	}

	if len(parts) == 1 {
		//this is the value we're looking for
		return namedValue
	}

	subObj, ok := namedValue.(map[string]interface{})
	if !ok {
		return nil
	}
	return jsonGet(subObj, parts[1])
} //jsonGet()
