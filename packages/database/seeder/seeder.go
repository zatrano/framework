package seeder

// Seeder seeds the database.
type Seeder interface {
	Run() error
}

// Runner runs seeders.
type Runner struct {
	seeders []Seeder
}

// NewRunner creates a seeder runner.
func NewRunner(seeders ...Seeder) *Runner {
	return &Runner{seeders: seeders}
}

// Call runs all registered seeders.
func (r *Runner) Call() error {
	for _, seeder := range r.seeders {
		if err := seeder.Run(); err != nil {
			return err
		}
	}
	return nil
}

// Register appends seeders.
func (r *Runner) Register(seeders ...Seeder) {
	r.seeders = append(r.seeders, seeders...)
}
