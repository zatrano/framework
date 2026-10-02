package doctor

import (
	"path/filepath"
	"testing"
)

func TestDoctorInteractorDirectoryFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "interactors", "create_post.go"), `package interactors

type CreatePostInteractor struct{}
`)
	assertDoctorRule(t, root, "APP-LAY-001")
	assertDoctorRule(t, root, "APP-LAY-002")
	assertDoctorRule(t, root, "APP-LAY-003")
}

func TestDoctorApplicationLayerDirectoryFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "application", "create_post.go"), `package application

func CreatePost() {}
`)
	assertDoctorRule(t, root, "APP-LAY-001")
}

func TestDoctorCoreDirectoryPasses(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "core", "clock.go"), `package core

func Now() {}
`)
	assertDoctorNoRule(t, root, "APP-LAY-001")
}

func TestDoctorWorkflowsDirectoryPasses(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "workflows", "note.go"), `package workflows

type Note struct{}
`)
	assertDoctorNoRule(t, root, "APP-LAY-001")
}

func TestDoctorControllerHelperTransactionFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "handlers", "web", "post_handler.go"), `package web

import "github.com/zatrano/packages/orm"

type PostHandler struct{}

func (c *PostHandler) Store() {
	runTx()
}

func runTx() {
	_ = orm.Transaction(func(tx any) error { return nil })
}
`)
	assertDoctorRule(t, root, "APP-CTL-005")
}

func TestDoctorTransactionInOtherPackageIsLimitation(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "handlers", "web", "post_handler.go"), `package web

type PostHandler struct{}

func (c *PostHandler) Store() {
	runAway()
}
`)
	writeDoctorFile(t, root, filepath.Join("app", "txutil", "tx.go"), `package txutil

import "github.com/zatrano/packages/orm"

func runAway() {
	_ = orm.Transaction(func(tx any) error { return nil })
}
`)
	assertDoctorNoRule(t, root, "APP-CTL-005")
}

func TestDoctorIndirectResponseHelperIsLimitation(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "handlers", "web", "post_handler.go"), `package web

import "github.com/zatrano/framework/v3/core/kernel/http"

type PostHandler struct{}

func (c *PostHandler) Show(req *http.Request) *http.Response {
	return renderShow(req)
}

func renderShow(req *http.Request) *http.Response {
	if req == nil {
		return http.JSON(map[string]any{})
	}
	return http.Template("posts/show", map[string]any{})
}
`)
	assertDoctorNoRule(t, root, "APP-CTL-003")
}

func TestDoctorAuthorControllerMixFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "handlers", "web", "author_handler.go"), `package web

import "github.com/zatrano/framework/v3/core/kernel/http"

type AuthorHandler struct{}

func (c *AuthorHandler) Show(req *http.Request) *http.Response {
	if req == nil {
		return http.JSON(map[string]any{})
	}
	return http.Template("authors/show", map[string]any{})
}
`)
	assertDoctorRule(t, root, "APP-CTL-003")
}

func TestDoctorPostFormRequestFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "requests", "post_form.go"), `package requests

type PostForm struct{}

func (PostForm) Rules() map[string]string {
	return map[string]string{"title": "required"}
}
`)
	assertDoctorRule(t, root, "APP-REQ-001")
}

func TestDoctorCreatePostRequestFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "requests", "create_post_request.go"), `package requests

type CreatePostRequest struct{}

func (CreatePostRequest) Rules() map[string]string {
	return map[string]string{"title": "required"}
}
`)
	assertDoctorRule(t, root, "APP-REQ-001")
}

func TestDoctorUnusedStoreRequestIsLimitation(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("bootstrap", "enabled.go"), `package bootstrap
var EnabledAddons = []string{}
`)
	writeDoctorFile(t, root, filepath.Join("app", "http", "requests", "post_store_request.go"), `package requests

type PostStoreRequest struct{}

func (PostStoreRequest) Rules() map[string]string {
	return map[string]string{"title": "required"}
}
`)
	writeDoctorFile(t, root, filepath.Join("app", "http", "handlers", "web", "post_handler.go"), `package web

import "github.com/zatrano/framework/v3/core/kernel/http"

type PostHandler struct{}

func (c *PostHandler) Store(req *http.Request) *http.Response {
	return http.Template("posts/create", map[string]any{})
}
`)
	assertDoctorNoRule(t, root, "APP-REQ-003")
}

func TestDoctorValidationMakeInServiceFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "services", "post.go"), `package services

import "github.com/zatrano/packages/validation"

func Create() {
	validation.Make(nil, map[string]string{"title": "required"})
}
`)
	assertDoctorRule(t, root, "APP-REQ-002")
}

func TestDoctorRepositoryInterfaceFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "repositories", "post_repository.go"), `package repositories

type PostRepository interface {
	Find(id uint) error
}
`)
	assertDoctorRule(t, root, "APP-REP-001")
}

func TestDoctorBaseRepositoryFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "repositories", "base.go"), `package repositories

type BaseRepository struct{}
type GenericRepository struct{}
type PostRepositoryFactory struct{}
`)
	assertDoctorRule(t, root, "APP-REP-001")
}

func TestDoctorConcreteRepositoryPasses(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "repositories", "post_repository.go"), `package repositories

import "github.com/zatrano/packages/orm"

type PostRepository struct{}

func (PostRepository) Find() {
	_ = orm.Query[any]()
}
`)
	assertDoctorNoRule(t, root, "APP-REP-001")
}

func TestDoctorHTTPHandlerTypePasses(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "handlers", "web", "post_handler.go"), `package web

import "github.com/zatrano/framework/v3/core/kernel/http"

type PostHandler struct{}

func (h *PostHandler) Store(req *http.Request) *http.Response {
	return http.Template("posts/show", map[string]any{})
}
`)
	assertDoctorNoRule(t, root, "APP-CTL-001")
}

func TestDoctorHTTPControllerTypeFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "handlers", "web", "post_handler.go"), `package web

import "github.com/zatrano/framework/v3/core/kernel/http"

type PostController struct{}

func (c *PostController) Store(req *http.Request) *http.Response {
	return http.Template("posts/show", map[string]any{})
}
`)
	assertDoctorRule(t, root, "APP-CTL-001")
}

func TestDoctorLogWithStringDoesNotFlagORM(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "services", "post.go"), `package services

import (
	"github.com/zatrano/packages/orm"
)

type logger struct{}

func (logger) With(msg string) {}

func Load(log logger) {
	log.With("comments")
	_ = orm.Query[any]()
}
`)
	assertDoctorNoRule(t, root, "APP-ORM-001")
}

func TestDoctorLogInfoCommentsPasses(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "services", "post.go"), `package services

import "github.com/zatrano/packages/orm"

func Load() {
	info("comments")
	_ = orm.Query[any]()
}

func info(msg string) {}
`)
	assertDoctorNoRule(t, root, "APP-ORM-001")
}

func TestDoctorQueryVariableWithIsLimitation(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "services", "post.go"), `package services

import "github.com/zatrano/packages/orm"

func Load() {
	q := orm.Query[any]()
	q.With("comments")
}
`)
	assertDoctorNoRule(t, root, "APP-ORM-001")
}

func TestDoctorRouterAssignedPutIsLimitation(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "providers", "routes.go"), `package providers

type app struct{}

func (app) Router() *router { return nil }

type router struct{}

func (*router) Put(path string, h any) {}

func Boot(a app) {
	r := a.Router()
	r.Put("/x", nil)
}
`)
	assertDoctorNoRule(t, root, "APP-ROUTE-002")
}

func TestDoctorAuthSurfaceRoutesPass(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "routes", "auth", "web", "auth.go"), `package web

func init() {
	router.Get("/login", nil)
}
`)
	writeDoctorFile(t, root, filepath.Join("app", "routes", "auth", "api", "auth.go"), `package api

func init() {
	router.Post("/login", nil)
}
`)
	writeDoctorFile(t, root, filepath.Join("app", "http", "handlers", "auth", "web", "auth_handler.go"), `package web

type AuthHandler struct{}
`)
	writeDoctorFile(t, root, filepath.Join("app", "http", "handlers", "auth", "api", "auth_handler.go"), `package api

type AuthHandler struct{}
`)
	findings := mustDoctor(t, root)
	if hasDoctorRule(findings, "APP-ROUTE-001") || hasDoctorRule(findings, "APP-CTL-001") {
		t.Fatalf("auth surface must pass doctor:\n%s", FormatDoctorText(root, findings))
	}
}

func TestDoctorMisplacedRouteRegistrationFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "routes", "web.go"), `package routes

func init() {
	router.Get("/posts", nil)
}
`)
	assertDoctorRule(t, root, "APP-ROUTE-001")
}

func TestDoctorFalsePositiveUnusualNames(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "models", "domain_event.go"), `package models

type DomainEvent struct{}
type LegalEntity struct{}
`)
	writeDoctorFile(t, root, filepath.Join("app", "http", "middleware", "action_log.go"), `package middleware

type ActionLog struct{}
`)
	writeDoctorFile(t, root, filepath.Join("app", "jobs", "notify_job.go"), `package jobs

type NotifyJob struct{}
`)
	writeDoctorFile(t, root, filepath.Join("app", "core", "processor.go"), `package core

type Workflow struct{}
type Processor struct{}
`)
	writeDoctorFile(t, root, filepath.Join("app", "services", "domain.go"), `package services

type Domain struct{}
`)
	findings := mustDoctor(t, root)
	for _, id := range []string{"APP-LAY-001", "APP-LAY-002", "APP-LAY-003", "APP-CTL-001", "APP-REP-001"} {
		if hasDoctorRule(findings, id) {
			t.Fatalf("false positive %s:\n%s", id, FormatDoctorText(root, findings))
		}
	}
}

func TestDoctorProductCRUDVariants(t *testing.T) {
	t.Run("controller_orm", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFile(t, root, filepath.Join("app", "http", "handlers", "web", "product_handler.go"), `package web

import (
	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/packages/orm"
)

type ProductHandler struct{}

func (c *ProductHandler) Index(req *http.Request) *http.Response {
	_ = orm.Query[any]()
	return http.Template("products/index", map[string]any{})
}
`)
		assertDoctorNoRule(t, root, "APP-LAY-003")
		assertDoctorNoRule(t, root, "APP-CTL-005")
	})
	t.Run("controller_service_orm", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFile(t, root, filepath.Join("app", "http", "handlers", "web", "product_handler.go"), `package web

import "github.com/zatrano/framework/v3/core/kernel/http"

type ProductHandler struct{}

func (c *ProductHandler) Store(req *http.Request) *http.Response {
	(&ProductService{}).Create()
	return http.Template("products/show", map[string]any{})
}
`)
		writeDoctorFile(t, root, filepath.Join("app", "services", "product.go"), `package web

import "github.com/zatrano/packages/orm"

type ProductService struct{}

func (s *ProductService) Create() {
	_, _ = orm.Create[any](map[string]any{"name": "x"})
}
`)
		assertDoctorNoRule(t, root, "APP-LAY-003")
		assertDoctorNoRule(t, root, "APP-CTL-005")
	})
	t.Run("controller_concrete_repo", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFile(t, root, filepath.Join("app", "http", "handlers", "web", "product_handler.go"), `package web

import "github.com/zatrano/framework/v3/core/kernel/http"

type ProductHandler struct{}

func (c *ProductHandler) Show(req *http.Request) *http.Response {
	_ = ProductRepository{}.Find()
	return http.Template("products/show", map[string]any{})
}
`)
		writeDoctorFile(t, root, filepath.Join("app", "repositories", "product_repository.go"), `package web

import "github.com/zatrano/packages/orm"

type ProductRepository struct{}

func (ProductRepository) Find() any {
	return orm.Query[any]()
}
`)
		assertDoctorNoRule(t, root, "APP-REP-001")
	})
	t.Run("controller_handler_http", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFile(t, root, filepath.Join("app", "http", "handlers", "web", "product_handler.go"), `package web

import "github.com/zatrano/framework/v3/core/kernel/http"

type ProductHandler struct{}

func (h *ProductHandler) Store(req *http.Request) *http.Response {
	return http.Template("products/show", map[string]any{})
}
`)
		assertDoctorNoRule(t, root, "APP-CTL-001")
	})
	t.Run("controller_suffix_http", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFile(t, root, filepath.Join("app", "http", "handlers", "web", "product_handler.go"), `package web

import "github.com/zatrano/framework/v3/core/kernel/http"

type ProductController struct{}

func (c *ProductController) Store(req *http.Request) *http.Response {
	return http.Template("products/show", map[string]any{})
}
`)
		assertDoctorRule(t, root, "APP-CTL-001")
	})
	t.Run("controller_usecase", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFile(t, root, filepath.Join("app", "services", "create_product.go"), `package services

type CreateProductUseCase struct{}
`)
		assertDoctorRule(t, root, "APP-LAY-003")
	})
}

func TestDoctorGoldenOrderUseCaseMutationFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "handlers", "web", "order_handler.go"), `package web

import "github.com/zatrano/framework/v3/core/kernel/http"

type OrderHandler struct{}

func (c *OrderHandler) Store(req *http.Request) *http.Response {
	_ = (&PlaceOrderUseCase{}).Place()
	return http.Template("orders/show", map[string]any{})
}
`)
	writeDoctorFile(t, root, filepath.Join("app", "services", "place_order_use_case.go"), `package web

import "github.com/zatrano/packages/orm"

type PlaceOrderUseCase struct{}

func (s *PlaceOrderUseCase) Place() error {
	return orm.Transaction(func(tx any) error { return nil })
}
`)
	assertDoctorRule(t, root, "APP-LAY-003")
}

func TestDoctorWebJSONAllowed(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "handlers", "web", "home_handler.go"), `package web

import "github.com/zatrano/framework/v3/core/kernel/http"

type HomeHandler struct{}

func (c *HomeHandler) Index(req *http.Request) *http.Response {
	return http.JSON(map[string]any{"ok": true})
}
`)
	assertDoctorNoRule(t, root, "APP-CTL-004")
	assertDoctorNoRule(t, root, "APP-CTL-003")
}

func TestDoctorUniqueConcatIsLimitation(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("bootstrap", "enabled.go"), `package bootstrap
var EnabledAddons = []string{"validation"}
`)
	writeDoctorFile(t, root, filepath.Join("app", "http", "requests", "user_store_request.go"), `package requests

type UserStoreRequest struct{}

func (UserStoreRequest) Rules() map[string]string {
	rule := "uni" + "que:users,email"
	return map[string]string{"email": rule}
}
`)
	assertDoctorNoRule(t, root, "APP-VAL-001")
}

func assertDoctorNoRule(t *testing.T, root, rule string) {
	t.Helper()
	findings := mustDoctor(t, root)
	if hasDoctorRule(findings, rule) {
		t.Fatalf("did not expect rule %s:\n%s", rule, FormatDoctorText(root, findings))
	}
}
