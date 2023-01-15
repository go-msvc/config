# Config #

A golang package to use configuration. Why?

Features:
* Validates configuration with easy to understand error messages.
* Can documents your configuration schema and values in use.
* Supports multiple and custom sources of configuration values.
* Supports construction of configured items.

# Construction of Items #

The most useful part of this library is to write code without knowing which implementation you are using. For example you need to host a micro-service, you can serve it as an HTTP REST API,
or a NATS or Kafka or RabbitMQ consumer, or run it against a file containing the request data.

In this case, your program can indicate the interface it needs and use the interface.

After completing the program, you can implement that interface in many ways, supporting all
the kinds of ways you want to serve the micro-service and then just configure the one to use.






Configuration is considered constant for the lifetime of your program.
If you need dynamic values, use a database.



Loads configuration.

Configuration:
- is constant, i.e. cannot change at run-time.
- is loaded once before it is used the first time.
- can be validated
- can be documents
- is name-value pairs, but the value can be complex

## Constructors ##
Constructors are supported to use configurable implementations of an interface.

To use this, you do the following:
- Define the interface
- Define a config that creates an object that implements the interface
- Register the config with a name
- Define more configs to implement the interface on other ways and register them too.
- Configure and use the config

Say you created a package called `github.com/myname/things` and you need a manager to get and update things, but this manager could be implemented in many ways, e.g. store in AWS S3 or in mongo or a simple local file or may be call some external HTTP API! The point is, you want the user ultimately to be able to change the implementation without updating your code.

Start in things package with a `manager.go` file to define the interface of a Manager:
```
package things

type Manager interface {
    Add(data interface{}) (Thing, error)
    Get(id string) (Thing,error)
    Put(Thing) error
    Del(id string) error
    //...
}
```

In another file in the same package called `thing.go` you define the Thing interface:
```
package things

type Thing interface {
    ID() string
    Value() interface{}
}
```

Now you start with your first implementation and to keep it simple, you decide to use local files. The files will be stored in a directory, so you need the directory to be configurable. Let's assume you put this in your own repo at `github.com/myname/filethings`

```
package filethings

//Config for the file thing manager
type Config struct {
    Directory string `json:"directory"
}

//Config implements config.Validator:
func (c Config) Validate() error {
    ... you will want to check the directory exists and you have write access
}

//Config also implements config.Constructor which will be called after config was validated
//and it must create the interface defined as things.Manager
func (c Config) Create() (things.Manager, error) {
    ...
    return manager{
        config: c,
    },nil
}

///register your config in your init func using a name that will identify the implementation
//against others, e.g. "files" tells you this manager manages things stored in files, compared
//to another may be called "s3" or "http" or "redis" etc...
func init() {
    ms.Register("files", Config{})
}

//you also need to implement the Manager interface on your manager{} type, i.e. define all
//this methods...
type manager {
    things.Manager
    config Config   //you would most likely want to store the config to use later
    ... more values  //you may have more values, e.g. cached values or connections etc.
}

func (mgr manager) Add(data interface{}) (Thing, error) {...}
func (mgr manager) Get(id string) (Thing,error) {...}
func (mgr manager) Put(Thing) error {...}
func (mgr manager) Del(id string) error {...}
//...
```

Now the thing manager can be use in another program as follows:
```
package main

import (
    //anonymous import of all implementations you want to support
    //when a new implementation becomes available, add it here
    _ "github.com/myname/filethings"
)

func main() {
    ...
}
```

You can put config in a JSON file in the same directory, or use another source.
You config would be something like this:
```
{
    "manager":{
        "files":{
            "directory":"./things"    
        }
    }
}
```

The manager can be created in your main or another function - but not in init() functions, because that might miss the registration of implementations. It is created like this:

```
mgr,err := config.Get("things.Manager")

//use your item
if t,err := mgr.Add(...); err != nil {
    ...
}
```

Now you can create more implementations, import into your main, compile and run with new config.
