package gen

import (
	"fmt"
	"os"
	"path/filepath"
)

// Job generates a job stub under baseDir/jobs/<name>.go.
func Job(moduleRoot, baseDir, rawName string, dryRun bool) ([]string, error) {
	name := normalizeName(rawName)
	if name == "" {
		return nil, fmt.Errorf("invalid job name %q (use letters, digits, _ or -)", rawName)
	}
	modPath, err := ModuleImportPath(moduleRoot)
	if err != nil {
		return nil, err
	}
	pascal := snakeToPascal(name)
	jobDir := filepath.Join(baseDir, "jobs")
	fileName := name + ".go"
	body := tmplJob(name, pascal, modPath)

	path := filepath.Join(jobDir, fileName)
	var written []string
	written = append(written, path)

	if dryRun {
		return written, nil
	}
	if err := os.MkdirAll(jobDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return nil, err
	}
	return written, nil
}

func tmplJob(pkg, pascal, modImport string) string {
	return fmt.Sprintf(`package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/zatrano/framework/pkg/queue"
)

// %[2]sJob handles the %[1]s background task.
// Register it with the queue manager at application startup:
//
//	queueManager.Register("%[1]s", func() queue.Job { return &%[2]sJob{} })
//
// Dispatch it:
//
//	queueManager.Dispatch(ctx, &jobs.%[2]sJob{
//	    // set fields here
//	})
type %[2]sJob struct {
	queue.BaseJob
	// Add your job-specific fields here.
	// Example:
	// UserID uint   `+"`"+`json:"user_id"`+"`"+`
	// Email  string `+"`"+`json:"email"`+"`"+`
}

// Name returns the unique identifier for this job type.
func (j *%[2]sJob) Name() string { return "%[1]s" }

// Queue returns the queue name (override BaseJob default).
func (j *%[2]sJob) Queue() string { return "default" }

// Retries returns the maximum number of retry attempts.
func (j *%[2]sJob) Retries() int { return 3 }

// Timeout returns the maximum execution duration.
func (j *%[2]sJob) Timeout() time.Duration { return 60 * time.Second }

// Handle contains the job logic. Return an error to trigger a retry.
func (j *%[2]sJob) Handle(ctx context.Context) error {
	fmt.Println("processing %[1]s job")
	// TODO: implement your job logic here.
	return nil
}
`, pkg, pascal, modImport)
}
