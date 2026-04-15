package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"

	"github.com/zatrano/framework/pkg/cache"
	"github.com/zatrano/framework/pkg/config"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Cache management commands",
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all cache entries or specific tags",
	Long: `Removes cache entries from the configured cache driver.

By default, clears all cache entries. Use --tag to clear only keys associated
with specific tags (requires Redis driver).

Examples:
  zatrano cache clear                  # clear all cache
  zatrano cache clear --tag users      # clear "users" tag only
  zatrano cache clear --tag users --tag posts  # clear multiple tags`,
	RunE: runCacheClear,
}

func init() {
	cacheClearCmd.Flags().StringSlice("tag", nil, "clear only keys with these tags (repeat for multiple)")
	cacheClearCmd.Flags().String("env", "", "environment profile (default $ZATRANO_ENV or dev)")
	cacheClearCmd.Flags().String("config-dir", "config", "config directory")
	cacheClearCmd.Flags().Bool("no-dotenv", false, "skip .env loading")
	cacheCmd.AddCommand(cacheClearCmd)
	rootCmd.AddCommand(cacheCmd)
}

func runCacheClear(cmd *cobra.Command, _ []string) error {
	tags, _ := cmd.Flags().GetStringSlice("tag")
	envName, _ := cmd.Flags().GetString("env")
	cfgDir, _ := cmd.Flags().GetString("config-dir")
	noDotenv, _ := cmd.Flags().GetBool("no-dotenv")

	cfg, err := config.Load(config.LoadOptions{
		Env:       envName,
		ConfigDir: cfgDir,
		DotEnv:    !noDotenv,
	})
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	redisURL := strings.TrimSpace(cfg.RedisURL)
	if redisURL == "" {
		// No Redis — use memory driver (only useful for testing).
		drv := cache.NewMemoryDriver()
		mgr := cache.New(drv)
		if err := mgr.Flush(ctx); err != nil {
			return fmt.Errorf("cache flush: %w", err)
		}
		fmt.Println("✓ Memory cache cleared.")
		return nil
	}

	// Redis driver.
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("redis url: %w", err)
	}
	client := redis.NewClient(opt)
	defer func() {
		_ = client.Close()
	}()

	drv := cache.NewRedisDriver(client)
	mgr := cache.New(drv)

	if len(tags) > 0 {
		// Clear specific tags.
		for _, tag := range tags {
			if err := cache.FlushTag(ctx, mgr, tag); err != nil {
				fmt.Fprintf(os.Stderr, "✗ tag %q: %v\n", tag, err)
				continue
			}
			fmt.Printf("✓ Tag %q flushed.\n", tag)
		}
		return nil
	}

	// Clear all cache.
	if err := mgr.Flush(ctx); err != nil {
		return fmt.Errorf("cache flush: %w", err)
	}
	fmt.Println("✓ All cache cleared.")
	return nil
}
