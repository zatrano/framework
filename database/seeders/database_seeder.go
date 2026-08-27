package seeders

import (
	"fmt"

	"github.com/zatrano/framework/app/models"
	"github.com/zatrano/framework/packages/database/seeder"
	"github.com/zatrano/framework/packages/hashing"
	"github.com/zatrano/framework/packages/orm"
)

// All returns application seeders.
func All() []seeder.Seeder {
	return []seeder.Seeder{
		&AdminUserSeeder{},
	}
}

// AdminUserSeeder creates a default dashboard admin when missing.
type AdminUserSeeder struct{}

func (s *AdminUserSeeder) Run() error {
	existing, err := orm.Query[models.User]().Where("email", "admin@zatrano.test").First()
	if err == nil && existing != nil {
		if !existing.IsAdmin {
			existing.IsAdmin = true
			return orm.Save(existing)
		}
		return nil
	}
	hash, err := hashing.Hash("password")
	if err != nil {
		return err
	}
	admin := &models.User{
		Name:     "Admin",
		Email:    "admin@zatrano.test",
		Password: hash,
		IsAdmin:  true,
	}
	if err := orm.Save(admin); err != nil {
		return fmt.Errorf("admin seeder: %w", err)
	}
	return nil
}
