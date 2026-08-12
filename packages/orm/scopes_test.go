package orm_test

import (
	"testing"

	"github.com/zatrano/framework/packages/orm"
)

type scopeUser struct {
	orm.Model
	Name   string `db:"name"`
	Active bool   `db:"active"`
}

func (scopeUser) TableName() string { return "scope_users_test" }

func TestLocalScopesHelpers(t *testing.T) {
	q := orm.Query[scopeUser]()
	called := false
	q = q.Scope(func(query *orm.Querier[scopeUser]) *orm.Querier[scopeUser] {
		called = true
		return query.Where("active", true)
	})
	_ = q.When(true, func(query *orm.Querier[scopeUser]) *orm.Querier[scopeUser] {
		return query.Where("name", "Ada")
	})
	if !called {
		t.Fatal("expected scope to run")
	}
}

func TestGlobalScopeRegistry(t *testing.T) {
	orm.ClearGlobalScopes[scopeUser]()
	orm.AddGlobalScope[scopeUser]("active", func(q *orm.Querier[scopeUser]) *orm.Querier[scopeUser] {
		return q.Where("active", true)
	})
	q := orm.Query[scopeUser]().WithoutGlobalScope("active")
	if q == nil {
		t.Fatal("expected querier")
	}
	orm.ClearGlobalScopes[scopeUser]()
}

func TestNamedLocalScope(t *testing.T) {
	orm.ClearLocalScopes[scopeUser]()
	orm.RegisterScope[scopeUser]("admins", func(q *orm.Querier[scopeUser]) *orm.Querier[scopeUser] {
		return q.Where("role", "admin")
	})
	if !orm.HasScope[scopeUser]("admins") {
		t.Fatal("HasScope")
	}
	q := orm.Query[scopeUser]().NamedScope("admins")
	if q == nil {
		t.Fatal("named")
	}
	orm.ClearLocalScopes[scopeUser]()
}
