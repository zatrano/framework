package migrations

import "github.com/zatrano/framework/core/database/migration"

// All returns application migrations in order.
func All() []migration.Migration {
	return []migration.Migration{
		&CreateJobsTable{},
		&CreateNotificationsTable{},
		&AddNotifiableTypeToNotifications{},
	}
}
