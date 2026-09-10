package console

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
	writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "post_controller.go"), `package web

import "github.com/zatrano/packages/orm"

type PostController struct{}

func (c *PostController) Store() {
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
	writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "post_controller.go"), `package web

type PostController struct{}

func (c *PostController) Store() {
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
	writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "post_controller.go"), `package web

import "github.com/zatrano/framework/v2/kernel/http"

type PostController struct{}

func (c *PostController) Show(req *http.Request) *http.Response {
	return renderShow(req)
}

func renderShow(req *http.Request) *http.Response {
	if req == nil {
		return http.JSON(map[string]any{})
	}
	return http.View("posts/show", map[string]any{})
}
`)
	assertDoctorNoRule(t, root, "APP-CTL-003")
}

func TestDoctorAuthorControllerMixFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "author_controller.go"), `package web

import "github.com/zatrano/framework/v2/kernel/http"

type AuthorController struct{}

func (c *AuthorController) Show(req *http.Request) *http.Response {
	if req == nil {
		return http.JSON(map[string]any{})
	}
	return http.View("authors/show", map[string]any{})
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
	writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "post_controller.go"), `package web

import "github.com/zatrano/framework/v2/kernel/http"

type PostController struct{}

func (c *PostController) Store(req *http.Request) *http.Response {
	return http.View("posts/create", map[string]any{})
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

func TestDoctorHTTPHandlerTypeFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "post_handler.go"), `package web

import "github.com/zatrano/framework/v2/kernel/http"

type PostHandler struct{}

func (h *PostHandler) Store(req *http.Request) *http.Response {
	return http.View("posts/show", map[string]any{})
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
	writeDoctorFile(t, root, filepath.Join("app", "jobs", "notify_handler.go"), `package jobs

type NotifyHandler struct{}
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
		writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "product_controller.go"), `package web

import (
	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/packages/orm"
)

type ProductController struct{}

func (c *ProductController) Index(req *http.Request) *http.Response {
	_ = orm.Query[any]()
	return http.View("products/index", map[string]any{})
}
`)
		assertDoctorNoRule(t, root, "APP-LAY-003")
		assertDoctorNoRule(t, root, "APP-CTL-005")
	})
	t.Run("controller_service_orm", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "product_controller.go"), `package web

import "github.com/zatrano/framework/v2/kernel/http"

type ProductController struct{}

func (c *ProductController) Store(req *http.Request) *http.Response {
	(&ProductService{}).Create()
	return http.View("products/show", map[string]any{})
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
		writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "product_controller.go"), `package web

import "github.com/zatrano/framework/v2/kernel/http"

type ProductController struct{}

func (c *ProductController) Show(req *http.Request) *http.Response {
	_ = ProductRepository{}.Find()
	return http.View("products/show", map[string]any{})
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
		writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "product_handler.go"), `package web

import "github.com/zatrano/framework/v2/kernel/http"

type ProductHandler struct{}

func (h *ProductHandler) Store(req *http.Request) *http.Response {
	return http.View("products/show", map[string]any{})
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
	writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "order_controller.go"), `package web

import "github.com/zatrano/framework/v2/kernel/http"

type OrderController struct{}

func (c *OrderController) Store(req *http.Request) *http.Response {
	_ = (&PlaceOrderUseCase{}).Place()
	return http.View("orders/show", map[string]any{})
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
	writeDoctorFile(t, root, filepath.Join("app", "http", "controllers", "web", "home_controller.go"), `package web

import "github.com/zatrano/framework/v2/kernel/http"

type HomeController struct{}

func (c *HomeController) Index(req *http.Request) *http.Response {
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
