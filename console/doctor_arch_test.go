package console

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorForbiddenUseCaseDirectory(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "usecases", "create_post.go"), `package usecases

type CreatePostUseCase struct{}
`)
	assertDoctorRule(t, root, "APP-LAY-001")
	assertDoctorRule(t, root, "APP-LAY-002")
	assertDoctorRule(t, root, "APP-LAY-003")
	assertDoctorFailsCLI(t, root)
}

func TestDoctorControllerTransactionFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "post_controller.go"), `package web

import "github.com/zatrano/packages/orm"

type PostController struct{}

func (c *PostController) Store() {
	_ = orm.Transaction(func(tx any) error { return nil })
}
`)
	assertDoctorRule(t, root, "APP-CTL-005")
}

func TestDoctorBarePostRequestFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "requests", "post_request.go"), `package requests

type PostRequest struct{}

func (PostRequest) Rules() map[string]string {
	return map[string]string{"title": "required"}
}
`)
	assertDoctorRule(t, root, "APP-REQ-001")
}

func TestDoctorStoreRequestNamePasses(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "requests", "post_store_request.go"), `package requests

type PostStoreRequest struct{}

func (PostStoreRequest) Rules() map[string]string {
	return map[string]string{"title": "required"}
}
`)
	findings := mustDoctor(t, root)
	if hasDoctorRule(findings, "APP-REQ-001") {
		t.Fatalf("PostStoreRequest must pass:\n%s", FormatDoctorText(root, findings))
	}
}

func TestDoctorMixedViewJSONFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "post_controller.go"), `package web

import "github.com/zatrano/framework/v2/kernel/http"

type PostController struct{}

func (c *PostController) Show(req *http.Request) *http.Response {
	if req == nil {
		return http.JSON(map[string]any{})
	}
	return http.View("posts/show", map[string]any{})
}
`)
	assertDoctorRule(t, root, "APP-CTL-003")
}

func TestDoctorAuthControllerMixPasses(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "auth_controller.go"), `package web

import "github.com/zatrano/framework/v2/kernel/http"

type AuthController struct{}

func (c *AuthController) Login(req *http.Request) *http.Response {
	if req == nil {
		return http.JSON(map[string]any{})
	}
	return http.View("auth/login", map[string]any{})
}
`)
	findings := mustDoctor(t, root)
	if hasDoctorRule(findings, "APP-CTL-003") {
		t.Fatalf("AuthController mix is the ADR-0009 exception:\n%s", FormatDoctorText(root, findings))
	}
}

func TestDoctorStringEagerFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "post_controller.go"), `package web

import "github.com/zatrano/packages/orm"

type PostController struct{}

func (c *PostController) Index() {
	orm.Query[any]().With("comments")
}
`)
	assertDoctorRule(t, root, "APP-ORM-001")
}

func TestDoctorTypedEagerPasses(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "services", "post.go"), `package services

import "github.com/zatrano/packages/orm"

func Load() {
	orm.Query[any]().With(orm.EagerHasMany[any, any]("Comments", "post_id"))
}
`)
	findings := mustDoctor(t, root)
	if hasDoctorRule(findings, "APP-ORM-001") {
		t.Fatalf("typed With must pass:\n%s", FormatDoctorText(root, findings))
	}
}

func TestDoctorUniqueWithoutDatabaseFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("bootstrap", "enabled.go"), `package bootstrap
var EnabledAddons = []string{"validation"}
`)
	writeDoctorFile(t, root, filepath.Join("app", "http", "requests", "user_store_request.go"), `package requests

type UserStoreRequest struct{}

func (UserStoreRequest) Rules() map[string]string {
	return map[string]string{"email": "required|unique:users,email"}
}
`)
	assertDoctorRule(t, root, "APP-VAL-001")
}

func TestDoctorServiceTransactionPasses(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "services", "order_placement_service.go"), `package services

import "github.com/zatrano/packages/orm"

type OrderPlacementService struct{}

func (s *OrderPlacementService) Place() error {
	return orm.Transaction(func(tx any) error { return nil })
}
`)
	findings := mustDoctor(t, root)
	if hasDoctorRule(findings, "APP-CTL-005") {
		t.Fatalf("service TX must pass:\n%s", FormatDoctorText(root, findings))
	}
}

func TestDoctorFalsePositiveBusinessNames(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "models", "domain_event.go"), `package models

type DomainEvent struct{}
type LegalEntity struct{}
`)
	writeDoctorFile(t, root, filepath.Join("app", "http", "middleware", "action_log.go"), `package middleware

type ActionLog struct{}
`)
	writeDoctorFile(t, root, filepath.Join("internal", "support", "helper.go"), `package support

func Help() {}
`)
	writeDoctorFile(t, root, filepath.Join("app", "jobs", "notify_handler.go"), `package jobs

type NotifyHandler struct{}
`)
	findings := mustDoctor(t, root)
	for _, id := range []string{"APP-LAY-001", "APP-LAY-002", "APP-LAY-003"} {
		if hasDoctorRule(findings, id) {
			t.Fatalf("false positive %s:\n%s", id, FormatDoctorText(root, findings))
		}
	}
}

func TestDoctorInlineValidationMakeFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "post_controller.go"), `package web

import "github.com/zatrano/packages/validation"

type PostController struct{}

func (c *PostController) Store() {
	validation.Make(nil, map[string]string{"title": "required"})
}
`)
	assertDoctorRule(t, root, "APP-REQ-002")
}

func TestDoctorPersistWithoutValidateFormFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("bootstrap", "enabled.go"), `package bootstrap
var EnabledAddons = []string{"validation"}
`)
	writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "post_controller.go"), `package web

import (
	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/packages/orm"
)

type PostController struct{}

func (c *PostController) Store(req *http.Request) *http.Response {
	_, _ = orm.Create[any](map[string]any{"title": "x"})
	return http.JSON(map[string]any{})
}
`)
	assertDoctorRule(t, root, "APP-REQ-003")
}

func TestDoctorAPIViewFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "api", "post_controller.go"), `package api

import "github.com/zatrano/framework/v2/kernel/http"

type PostController struct{}

func (c *PostController) Index(req *http.Request) *http.Response {
	return http.View("posts/index", map[string]any{})
}
`)
	assertDoctorRule(t, root, "APP-CTL-004")
}

func TestDoctorAppRouterPutFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "providers", "routes.go"), `package providers

type app struct{}

func (app) Router() *router { return nil }

type router struct{}

func (*router) Put(path string, h any) {}

func Boot(a app) {
	a.Router().Put("/x", nil)
}
`)
	assertDoctorRule(t, root, "APP-ROUTE-002")
}

func TestDoctorJSONOutputAndExit(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "handlers", "oops.go"), `package handlers
`)
	var buf bytes.Buffer
	cmd := &DoctorCommand{out: &buf}
	err := cmd.Handle([]string{root, "--json"})
	if err == nil {
		t.Fatal("expected architecture error")
	}
	if CodeFromError(err) != ExitGeneral {
		t.Fatalf("exit %d", CodeFromError(err))
	}
	if !strings.Contains(buf.String(), `"rule": "APP-LAY-001"`) && !strings.Contains(buf.String(), `"APP-LAY-001"`) {
		t.Fatalf("json missing rule:\n%s", buf.String())
	}
}

func TestDoctorWarningsDoNotFail(t *testing.T) {
	root := filepath.Join("testdata", "doctor", "concrete_leak")
	var buf bytes.Buffer
	cmd := &DoctorCommand{out: &buf}
	if err := cmd.Handle([]string{root}); err != nil {
		t.Fatalf("warnings must not fail: %v\n%s", err, buf.String())
	}
}

func TestDoctorStrictTreatsWarningsAsErrors(t *testing.T) {
	root := filepath.Join("testdata", "doctor", "concrete_leak")
	var buf bytes.Buffer
	cmd := &DoctorCommand{out: &buf}
	if err := cmd.Handle([]string{root, "--strict"}); err == nil {
		t.Fatalf("expected --strict failure\n%s", buf.String())
	}
}

func assertDoctorFailsCLI(t *testing.T, root string) {
	t.Helper()
	var buf bytes.Buffer
	cmd := &DoctorCommand{out: &buf}
	if err := cmd.Handle([]string{root}); err == nil {
		t.Fatalf("expected doctor error\n%s", buf.String())
	}
}

func mustDoctor(t *testing.T, root string) []Finding {
	t.Helper()
	findings, err := RunDoctor(root)
	if err != nil {
		t.Fatal(err)
	}
	return findings
}

func assertDoctorRule(t *testing.T, root, rule string) {
	t.Helper()
	findings := mustDoctor(t, root)
	if !hasDoctorRule(findings, rule) {
		t.Fatalf("expected rule %s, got:\n%s", rule, FormatDoctorText(root, findings))
	}
}

func hasDoctorRule(findings []Finding, rule string) bool {
	for _, f := range findings {
		if f.Rule == rule {
			return true
		}
	}
	return false
}
