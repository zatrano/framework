package migrations

import "github.com/zatrano/framework/packages/database/schema"

// AddNotifiableTypeToNotifications adds polymorphic type column when upgrading.
type AddNotifiableTypeToNotifications struct{}

func (m *AddNotifiableTypeToNotifications) Name() string {
	return "20260812_000004_add_notifiable_type_to_notifications"
}

func (m *AddNotifiableTypeToNotifications) Up(s *schema.Builder) error {
	ok, err := s.HasTable("notifications")
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	has, err := s.HasColumn("notifications", "notifiable_type")
	if err != nil {
		return err
	}
	if has {
		return nil
	}
	return s.Table("notifications", func(table *schema.Blueprint) {
		table.String("notifiable_type").Default("recipient")
	})
}

func (m *AddNotifiableTypeToNotifications) Down(s *schema.Builder) error {
	return nil
}
