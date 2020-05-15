package config

//ISource ...
type ISource interface {
	Name() string
	Get(name string, tmpl IData) (IData, error)
	GetAll(name string) map[string]interface{}
}
