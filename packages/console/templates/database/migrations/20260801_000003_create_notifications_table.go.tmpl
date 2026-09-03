package migrations

import "github.com/zatrano/framework/packages/database/schema"

// CreateNotificationsTable creates the notifications table.
type CreateNotificationsTable struct{}

func (m *CreateNotificationsTable) Name() string {
	return "20260801_000003_create_notifications_table"
}

func (m *CreateNotificationsTable) Up(s *schema.Builder) error {
	return s.Create("notifications", func(table *schema.Blueprint) {
		table.ID()
		table.String("notifiable_type").Default("recipient")
		table.String("notifiable_id")
		table.String("type")
		table.Text("data")
		table.Timestamp("read_at").Nullable()
		table.Timestamps()
	})
}

func (m *CreateNotificationsTable) Down(s *schema.Builder) error {
	return s.DropIfExists("notifications")
}
