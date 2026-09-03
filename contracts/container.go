package contracts

// Container is the service container surface used via Application.Container().
type Container interface {
	Instance(abstract string, instance any)
}
