package factories

import (
	"fmt"
	"time"

	"github.com/zatrano/framework/app/models"
	"github.com/zatrano/framework/packages/factory"
	"github.com/zatrano/framework/packages/hashing"
)

func init() {
	factory.For[models.User](func() map[string]any {
		password, _ := hashing.Hash(factory.FakePassword())
		n := factory.Sequence("users")
		return map[string]any{
			"name":     factory.FakeName(),
			"email":    fmt.Sprintf("user%d@example.test", n),
			"password": password,
		}
	})
	factory.RegisterState[models.User]("verified", func() map[string]any {
		now := time.Now().UTC()
		return map[string]any{
			"email_verified_at": now,
			"name":              "Verified " + factory.FakeName(),
		}
	})
}
