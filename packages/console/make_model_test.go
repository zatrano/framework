package console

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/kernel"
)

func TestMakeModelTranslationColumns(t *testing.T) {
	dir := t.TempDir()
	app := kernel.NewApplication(dir)
	cmd := &MakeModelCommand{app: app}
	if err := cmd.Handle([]string{"Product", "--translation"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "app", "models", "product.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(body)
	for _, want := range []string{`NameTr`, `name_tr`, `NameEn`, `name_en`, `products`} {
		if !strings.Contains(src, want) {
			t.Fatalf("missing %q in:\n%s", want, src)
		}
	}
	if strings.Contains(src, `Name string`) {
		t.Fatal("expected no plain Name field with --translation")
	}
}

func TestMakeModelTranslationJSON(t *testing.T) {
	dir := t.TempDir()
	app := kernel.NewApplication(dir)
	cmd := &MakeModelCommand{app: app}
	if err := cmd.Handle([]string{"Category", "--translation=json", "-m"}); err != nil {
		t.Fatal(err)
	}
	model, err := os.ReadFile(filepath.Join(dir, "app", "models", "category.go"))
	if err != nil {
		t.Fatal(err)
	}
	ms := string(model)
	if !strings.Contains(ms, `Translations map[string]string`) || !strings.Contains(ms, `"translations": "json"`) {
		t.Fatalf("bad model:\n%s", ms)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "database", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	var mig []byte
	for _, e := range entries {
		if strings.Contains(e.Name(), "create_categories_table") {
			mig, err = os.ReadFile(filepath.Join(dir, "database", "migrations", e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	if mig == nil {
		t.Fatal("migration not created")
	}
	mg := string(mig)
	if !strings.Contains(mg, `Create("categories"`) || !strings.Contains(mg, `Text("translations")`) {
		t.Fatalf("bad migration:\n%s", mg)
	}
	if strings.Contains(mg, "schema *schema.Builder") {
		t.Fatal("parameter must not shadow schema package")
	}
}

func TestMakeModelMigrationColumns(t *testing.T) {
	dir := t.TempDir()
	app := kernel.NewApplication(dir)
	cmd := &MakeModelCommand{app: app}
	if err := cmd.Handle([]string{"Tag", "-t", "-m"}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "database", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	var mig string
	for _, e := range entries {
		if strings.Contains(e.Name(), "create_tags_table") {
			b, err := os.ReadFile(filepath.Join(dir, "database", "migrations", e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			mig = string(b)
			break
		}
	}
	if mig == "" {
		t.Fatal("migration missing")
	}
	for _, want := range []string{`Create("tags"`, `name_tr`, `name_en`, `Up(s *schema.Builder)`} {
		if !strings.Contains(mig, want) {
			t.Fatalf("missing %q in:\n%s", want, mig)
		}
	}
}
