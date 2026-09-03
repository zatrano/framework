package contracts

// Logger is the application logger surface used via Application.Logger().
type Logger interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}
