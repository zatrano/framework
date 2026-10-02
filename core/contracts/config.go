package contracts

// ConfigRepository is the configuration surface used via Application.Config().
type ConfigRepository interface {
	Get(key string, fallback ...any) any
	GetString(key string, fallback ...string) string
	GetInt(key string, fallback ...int) int
	GetBool(key string, fallback ...bool) bool
	All() map[string]any
	Load(name string, values map[string]any)
}
