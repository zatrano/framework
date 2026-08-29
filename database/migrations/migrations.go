package migrations

import "github.com/zatrano/framework/packages/database/migration"

// All returns application migrations in order.
// Auth/dashboard tables are added by make:auth / make:dashboard.
func All() []migration.Migration {
	return []migration.Migration{
		&CreateJobsTable{},
		&CreateNotificationsTable{},
		&AddNotifiableTypeToNotifications{},
	}
}
