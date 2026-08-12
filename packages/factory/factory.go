package factory

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/orm"
)

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

// Definition returns attribute defaults for a model.
type Definition func() map[string]any

var (
	mu          sync.RWMutex
	definitions = map[string]Definition{}
)

// Register registers a factory definition by name.
func Register(name string, definition Definition) {
	mu.Lock()
	defer mu.Unlock()
	definitions[name] = definition
}

// For registers a factory definition for a model type name.
func For[T any](definition Definition) {
	Register(typeName[T](), definition)
}

// definitionOf resolves a registered definition.
func definitionOf(name string) (Definition, error) {
	mu.RLock()
	defer mu.RUnlock()
	def, ok := definitions[name]
	if !ok {
		return nil, fmt.Errorf("factory [%s] is not defined", name)
	}
	return def, nil
}

// Make builds attributes without persisting.
func Make[T any](overrides ...map[string]any) (map[string]any, error) {
	def, err := definitionOf(typeName[T]())
	if err != nil {
		return nil, err
	}
	attrs := def()
	if len(overrides) > 0 {
		for key, value := range overrides[0] {
			attrs[key] = value
		}
	}
	return attrs, nil
}

// MakeMany builds many attribute maps.
func MakeMany[T any](count int, overrides ...map[string]any) ([]map[string]any, error) {
	out := make([]map[string]any, 0, count)
	for i := 0; i < count; i++ {
		attrs, err := Make[T](overrides...)
		if err != nil {
			return nil, err
		}
		out = append(out, attrs)
	}
	return out, nil
}

// Create persists a model using ORM.
func Create[T any](overrides ...map[string]any) (*T, error) {
	attrs, err := Make[T](overrides...)
	if err != nil {
		return nil, err
	}
	return orm.Create[T](attrs)
}

// CreateMany persists many models.
func CreateMany[T any](count int, overrides ...map[string]any) ([]T, error) {
	out := make([]T, 0, count)
	for i := 0; i < count; i++ {
		model, err := Create[T](overrides...)
		if err != nil {
			return nil, err
		}
		out = append(out, *model)
	}
	return out, nil
}

// Fake helpers

// FakeName returns a random name.
func FakeName() string {
	names := []string{"Ada", "Grace", "Alan", "Edsger", "Barbara", "Linus", "Guido", "Rob"}
	return names[seededRand.Intn(len(names))]
}

// FakeEmail returns a random email.
func FakeEmail() string {
	return fmt.Sprintf("%s%d@example.test", Snake(FakeName()), seededRand.Intn(10000))
}

// FakePassword returns a plain demo password.
func FakePassword() string {
	return "password"
}

// Sequence returns an incrementing value for a key.
var sequences = map[string]int{}
var seqMu sync.Mutex

// Sequence increments and returns a counter for key.
func Sequence(key string) int {
	seqMu.Lock()
	defer seqMu.Unlock()
	sequences[key]++
	return sequences[key]
}

// Snake is a tiny helper for email generation.
func Snake(value string) string {
	out := make([]rune, 0, len(value))
	for _, r := range value {
		if r >= 'A' && r <= 'Z' {
			out = append(out, r+'a'-'A')
			continue
		}
		if r == ' ' {
			out = append(out, '-')
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

func typeName[T any]() string {
	var zero T
	return fmt.Sprintf("%T", zero)
}
