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
	} else {
		log.Debugf("got file(%s): %+v", name, f.json)
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
	log.Debugf("%s.Get(%s)...", files.Name(), name)

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
}

func (files *source) GetAll(name string) map[string]config.IData {
	log.Debugf("%s.GetAll(%s)...", files.Name(), name)
	return nil
}

type file struct {
	name string
	path string
	json map[string]interface{}
}

func (f file) Get(name string, tmpl config.IData) (config.IData, error) {
	log.Debugf("file(%s).Get(%s)...", f.name, name)
	if f.json == nil {
		return nil, nil
	}

	dataObj := jsonGet(f.json, name)
	if dataObj == nil {
		return nil, nil
	}
	log.Debugf("dataObj: %+v", dataObj)
	jsonObj, _ := json.Marshal(dataObj)
	log.Debugf("jsonObj: %+v", string(jsonObj))

	//got the data - parse into struct
	dataValue := reflect.New(reflect.TypeOf(tmpl))
	configData := dataValue.Interface().(config.IData)
	if err := json.Unmarshal(jsonObj, configData); err != nil {
		return nil, log.Wrapf(err, "file(%s): cannot decode %s into %T", f.name, name, configData)
	}
	if err := configData.Validate(); err != nil {
		return nil, log.Wrapf(err, "file(%s): invalid %s", f.name, name)
	}

	log.Debugf("loaded %s (%T):%+v", name, configData, configData)
	return configData, nil
}

func jsonGet(obj map[string]interface{}, name string) interface{} {
	log.Debugf("Getting json(%s): %+v", name, obj)
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
