package config

//IData is configuration data/values
type IData interface {
	Validate() error
}

//ILoaded is current config data that can be used ...
// type ILoaded interface {
// 	//name of this config
// 	Name() string
// 	//value of this config
// 	Data() IData
// }

// //NewLoaded ...
// func NewLoaded(name string, data IData) ILoaded {
// 	return loaded{
// 		name: name,
// 		data: data,
// 		subs: nil,
// 	}
// }

// type loaded struct {
// 	name string
// 	data IData
// 	subs map[string]ILoaded
// }

// func (l loaded) Name() string {
// 	return l.name
// }

// func (l loaded) Data() IData {
// 	return l.data
// }
