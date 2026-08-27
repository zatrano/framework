package migrations

import "github.com/zatrano/framework/packages/database/migration"

// All returns application migrations in order.
func All() []migration.Migration {
	return []migration.Migration{
		&CreateJobsTable{},
		&CreateNotificationsTable{},
		&AddNotifiableTypeToNotifications{},
		&CreateUsersTable{},
		&CreatePasswordResetTokensTable{},
		&CreatePersonalAccessTokensTable{},
		&CreateSocialAccountsTable{},
		&CreateDashboardRolesTables{},
		&CreateDashboardRBACTables{},
		&CreateDashboardSettingsTable{},
	}
}
