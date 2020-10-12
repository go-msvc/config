package config

//ISource ...
type ISource interface {
	Name() string
	Get(name string) (value interface{}, err error)
	//GetAll(name string) map[string]interface{}
}
