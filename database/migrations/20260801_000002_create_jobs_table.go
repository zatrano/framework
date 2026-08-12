package migrations

import "github.com/zatrano/framework/packages/database/schema"

// CreateJobsTable creates the jobs table.
type CreateJobsTable struct{}

func (m *CreateJobsTable) Name() string {
	return "20260801_000002_create_jobs_table"
}

func (m *CreateJobsTable) Up(s *schema.Builder) error {
	// Database queue EnsureTable may CREATE TABLE IF NOT EXISTS during bootstrap
	// before migrate runs; skip if the table is already present.
	if ok, err := s.HasTable("jobs"); err != nil {
		return err
	} else if ok {
		return nil
	}
	return s.Create("jobs", func(table *schema.Blueprint) {
		table.ID()
		table.String("queue").Default("default")
		table.Text("payload")
		table.Timestamp("available_at")
		table.Timestamp("created_at")
		table.Timestamp("reserved_at").Nullable()
	})
}

func (m *CreateJobsTable) Down(s *schema.Builder) error {
	return s.DropIfExists("jobs")
}
