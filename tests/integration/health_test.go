//go:build integration
// +build integration

// Integration testler gerçek DB ve Redis gerektirir.
// Çalıştırma: go test -tags=integration ./tests/integration/...
// CI/CD: docker-compose -f docker-compose.test.yml up --abort-on-container-exit
package integration_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/zatrano/framework/configs/databaseconfig"
	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/configs/redisconfig"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestApp(t *testing.T) *fiber.App {
	t.Helper()
	os.Setenv("APP_ENV", "test")
	envconfig.Load()
	logconfig.InitLogger()
	databaseconfig.InitDB()
	redisconfig.InitRedis()

	app := fiber.New(fiber.Config{
		// Test modunda loglama azalt
	})

	app.Get("/health", func(c fiber.Ctx) error {
		db, _ := databaseconfig.GetDB().DB()
		dbOk := db.Ping() == nil
		_, redisErr := redisconfig.GetClient().Ping(c).Result()
		redisOk := redisErr == nil
		status := 200
		if !dbOk || !redisOk {
			status = 503
		}
		return c.Status(status).JSON(fiber.Map{
			"ok": dbOk && redisOk, "database": dbOk, "redis": redisOk,
		})
	})

	t.Cleanup(func() {
		databaseconfig.CloseDB()
		redisconfig.Close()
	})
	return app
}

func TestHealth_Endpoint_Returns200(t *testing.T) {
	app := setupTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	require.NoError(t, err)
	assert.Equal(t, true, data["ok"])
	assert.Equal(t, true, data["database"])
	assert.Equal(t, true, data["redis"])
}
