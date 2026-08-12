package query_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/database/query"
)

func TestLockClauses(t *testing.T) {
	mysql := query.New(nil, "mysql", "posts")
	sqlStr, _ := mysql.Where("id", 1).LockForUpdate().SkipLocked().ToSQL()
	if !strings.Contains(sqlStr, "FOR UPDATE SKIP LOCKED") {
		t.Fatalf("mysql for update skip=%s", sqlStr)
	}

	pg := query.New(nil, "postgres", "posts")
	sqlStr, _ = pg.Where("id", 1).SharedLock().NoWait().ToSQL()
	if !strings.Contains(sqlStr, "FOR SHARE NOWAIT") {
		t.Fatalf("pg share nowait=%s", sqlStr)
	}

	sqlite := query.New(nil, "sqlite", "posts")
	sqlStr, _ = sqlite.Where("id", 1).ForUpdate().ToSQL()
	if strings.Contains(sqlStr, "FOR UPDATE") {
		t.Fatalf("sqlite should omit lock: %s", sqlStr)
	}
}
