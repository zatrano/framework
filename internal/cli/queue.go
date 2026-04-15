package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/zatrano/framework/pkg/config"
	"github.com/zatrano/framework/pkg/queue"
)

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Queue management commands",
}

// ─── queue work ────────────────────────────────────────────────────────────

var queueWorkCmd = &cobra.Command{
	Use:   "work",
	Short: "Start the queue worker process",
	Long: `Starts a long-running worker that processes jobs from the queue.

The worker polls Redis for jobs and executes them with automatic retry
and exponential backoff. Failed jobs are recorded in PostgreSQL.

Examples:
  zatrano queue work
  zatrano queue work --queue emails --queue notifications
  zatrano queue work --tries 5 --timeout 120s --sleep 5s`,
	RunE: runQueueWork,
}

// ─── queue failed ──────────────────────────────────────────────────────────

var queueFailedCmd = &cobra.Command{
	Use:   "failed",
	Short: "List failed jobs",
	Long:  "Lists all jobs that failed after exhausting their retry attempts.",
	RunE:  runQueueFailed,
}

// ─── queue retry ───────────────────────────────────────────────────────────

var queueRetryCmd = &cobra.Command{
	Use:   "retry [id]",
	Short: "Retry a failed job (or all with --all)",
	Long: `Re-dispatches a failed job back to the queue for processing.

Examples:
  zatrano queue retry 42       # retry failed job with ID 42
  zatrano queue retry --all    # retry all failed jobs`,
	RunE: runQueueRetry,
}

// ─── queue flush ───────────────────────────────────────────────────────────

var queueFlushCmd = &cobra.Command{
	Use:   "flush",
	Short: "Delete all failed jobs",
	Long:  "Permanently removes all records from the failed_jobs table.",
	RunE:  runQueueFlush,
}

func init() {
	// queue work flags
	queueWorkCmd.Flags().StringSlice("queue", []string{"default"}, "queues to process (repeat for multiple)")
	queueWorkCmd.Flags().Int("tries", 3, "maximum number of attempts per job")
	queueWorkCmd.Flags().Duration("timeout", 60*time.Second, "default job execution timeout")
	queueWorkCmd.Flags().Duration("sleep", 3*time.Second, "sleep duration when no jobs available")

	// queue retry flags
	queueRetryCmd.Flags().Bool("all", false, "retry all failed jobs")

	// shared config flags
	for _, cmd := range []*cobra.Command{queueWorkCmd, queueFailedCmd, queueRetryCmd, queueFlushCmd} {
		cmd.Flags().String("env", "", "environment profile")
		cmd.Flags().String("config-dir", "config", "config directory")
		cmd.Flags().Bool("no-dotenv", false, "skip .env loading")
	}

	queueCmd.AddCommand(queueWorkCmd, queueFailedCmd, queueRetryCmd, queueFlushCmd)
	rootCmd.AddCommand(queueCmd)
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func loadQueueConfig(cmd *cobra.Command) (*config.Config, error) {
	envName, _ := cmd.Flags().GetString("env")
	cfgDir, _ := cmd.Flags().GetString("config-dir")
	noDotenv, _ := cmd.Flags().GetBool("no-dotenv")
	return config.Load(config.LoadOptions{
		Env:       envName,
		ConfigDir: cfgDir,
		DotEnv:    !noDotenv,
	})
}

func openRedisForQueue(cfg *config.Config) (*redis.Client, error) {
	u := strings.TrimSpace(cfg.RedisURL)
	if u == "" {
		return nil, fmt.Errorf("redis_url is required for queue operations")
	}
	opt, err := redis.ParseURL(u)
	if err != nil {
		return nil, fmt.Errorf("redis url: %w", err)
	}
	return redis.NewClient(opt), nil
}

func openDBForQueue(cfg *config.Config) (*gorm.DB, error) {
	u := strings.TrimSpace(cfg.DatabaseURL)
	if u == "" {
		return nil, nil // DB is optional
	}
	db, err := gorm.Open(postgres.Open(u), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	return db, nil
}

// ─── Implementations ──────────────────────────────────────────────────────

func runQueueWork(cmd *cobra.Command, _ []string) error {
	cfg, err := loadQueueConfig(cmd)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	client, err := openRedisForQueue(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	db, err := openDBForQueue(cfg)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	queues, _ := cmd.Flags().GetStringSlice("queue")
	tries, _ := cmd.Flags().GetInt("tries")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	sleep, _ := cmd.Flags().GetDuration("sleep")

	logger, _ := zap.NewProduction()

	drv := queue.NewRedisDriver(client)
	mgr := queue.New(drv)

	// NOTE: In a real application, you would register all your job types here.
	// mgr.Register("send_email", func() queue.Job { return &jobs.SendEmailJob{} })

	worker := queue.NewWorker(mgr, queue.WorkerConfig{
		Queues:   queues,
		MaxTries: tries,
		Timeout:  timeout,
		Sleep:    sleep,
		Logger:   logger,
		DB:       db,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT/SIGTERM.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		fmt.Println("\nShutting down worker...")
		cancel()
		worker.Stop()
	}()

	worker.Run(ctx)
	return nil
}

func runQueueFailed(cmd *cobra.Command, _ []string) error {
	cfg, err := loadQueueConfig(cmd)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	db, err := openDBForQueue(cfg)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}
	if db == nil {
		return fmt.Errorf("database_url is required to view failed jobs")
	}

	var jobs []queue.FailedJob
	if err := db.Order("failed_at DESC").Limit(100).Find(&jobs).Error; err != nil {
		return fmt.Errorf("query failed jobs: %w", err)
	}

	if len(jobs) == 0 {
		fmt.Println("No failed jobs found.")
		return nil
	}

	fmt.Printf("%-6s %-36s %-15s %-20s %s\n", "ID", "JOB_ID", "QUEUE", "JOB_NAME", "FAILED_AT")
	fmt.Println(strings.Repeat("-", 100))
	for _, j := range jobs {
		fmt.Printf("%-6d %-36s %-15s %-20s %s\n",
			j.ID, j.JobID, j.Queue, j.JobName, j.FailedAt.Format(time.RFC3339))
	}
	fmt.Printf("\nTotal: %d failed job(s)\n", len(jobs))
	return nil
}

func runQueueRetry(cmd *cobra.Command, args []string) error {
	retryAll, _ := cmd.Flags().GetBool("all")
	if !retryAll && len(args) == 0 {
		return fmt.Errorf("provide a failed job ID or use --all")
	}

	cfg, err := loadQueueConfig(cmd)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	client, err := openRedisForQueue(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	db, err := openDBForQueue(cfg)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}
	if db == nil {
		return fmt.Errorf("database_url is required to retry failed jobs")
	}

	drv := queue.NewRedisDriver(client)
	ctx := context.Background()

	var jobs []queue.FailedJob
	if retryAll {
		if err := db.Find(&jobs).Error; err != nil {
			return fmt.Errorf("query failed jobs: %w", err)
		}
	} else {
		var j queue.FailedJob
		if err := db.First(&j, args[0]).Error; err != nil {
			return fmt.Errorf("failed job %s not found: %w", args[0], err)
		}
		jobs = append(jobs, j)
	}

	if len(jobs) == 0 {
		fmt.Println("No failed jobs to retry.")
		return nil
	}

	for _, j := range jobs {
		if err := drv.Push(ctx, j.Queue, []byte(j.Payload)); err != nil {
			fmt.Fprintf(os.Stderr, "✗ job %d: %v\n", j.ID, err)
			continue
		}
		db.Delete(&j)
		fmt.Printf("✓ Job %d (%s) re-dispatched to queue %q\n", j.ID, j.JobName, j.Queue)
	}
	return nil
}

func runQueueFlush(cmd *cobra.Command, _ []string) error {
	cfg, err := loadQueueConfig(cmd)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	db, err := openDBForQueue(cfg)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}
	if db == nil {
		return fmt.Errorf("database_url is required to flush failed jobs")
	}

	result := db.Where("1 = 1").Delete(&queue.FailedJob{})
	if result.Error != nil {
		return fmt.Errorf("flush failed jobs: %w", result.Error)
	}
	fmt.Printf("✓ Deleted %d failed job(s).\n", result.RowsAffected)
	return nil
}
