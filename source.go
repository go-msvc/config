package config

//ISource ...
type ISource interface {
	Name() string

	//Get a value
	//if notifier is specified, it will be called once when the used value is
	//modified or deleted, and it should call Get again to get the new value
	//because the new value may be from a different source than the original
	//if the source does not implement run-time value deletion/changes, it can
	//ignore the notifier argument
	Get(name string, notifier INotifier) (value interface{}, err error)
}

type INotifier interface {
	Notify(name string) //no value... get it from all sources incase now from different source
}
