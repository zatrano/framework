package doctor

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

const archSee = "https://zatrano.com/docs/application-engineering/standard"

var forbiddenLayerDirs = []string{
	"domain",
	"app/domain",
	"dtos",
	"dto",
	"app/dtos",
	"app/dto",
	"usecases",
	"usecase",
	"app/usecases",
	"app/usecase",
	"interactors",
	"interactor",
	"app/interactors",
	"app/interactor",
	"app/application",
	"handlers",
	"app/handlers",
	"app/http/handlers",
	"actions",
	"app/actions",
	"app/http/actions",
	"entities",
	"app/entities",
	"internal/usecase",
	"internal/usecases",
	"internal/interactor",
	"internal/interactors",
	"internal/entity",
	"internal/entities",
	"internal/dto",
	"internal/dtos",
}

var forbiddenPackageNames = map[string]bool{
	"usecase": true, "usecases": true,
	"interactor": true, "interactors": true,
	"dto": true, "dtos": true,
	"handlers": true, "actions": true, "entities": true,
}

func checkForbiddenLayers(root string) ([]Finding, error) {
	var out []Finding
	for _, dir := range forbiddenLayerDirs {
		if st, err := os.Stat(filepath.Join(root, filepath.FromSlash(dir))); err != nil || !st.IsDir() {
			continue
		}
		out = append(out, Finding{
			Rule:     "APP-LAY-001",
			Check:    "layers",
			Severity: "error",
			File:     dir,
			Found:    "forbidden architectural directory " + dir,
			Why:      "STANDARD rejects domain/usecase/interactor/dto/handler/action/entity layers as application architecture.",
			How:      "Delete this directory and keep Controller → FormRequest → optional Service → ORM / From(app).",
			See:      archSee + " §B · ADR-0001",
		})
	}
	err := walkConsumerGo(root, func(rel, abs string, fset *token.FileSet, file *ast.File) {
		pkg := file.Name.Name
		if forbiddenPackageNames[pkg] || (pkg == "domain" && consumerAppPackage(rel)) {
			out = append(out, Finding{
				Rule:     "APP-LAY-002",
				Check:    "layers",
				Severity: "error",
				File:     rel,
				Line:     fset.Position(file.Name.Pos()).Line,
				Found:    "package " + pkg,
				Why:      "Package names usecase/interactor/dto/handlers/actions/entities/domain are a second architecture, not a ZATRANO application package.",
				How:      "Move types into app/services, app/models, or app/http/requests with canonical names.",
				See:      archSee + " §B · ADR-0001",
			})
		}
		ast.Inspect(file, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok || ts.Name == nil || !ts.Name.IsExported() {
				return true
			}
			name := ts.Name.Name
			if _, ok := ts.Type.(*ast.InterfaceType); ok && strings.HasSuffix(name, "Repository") {
				out = append(out, Finding{
					Rule:     "APP-REP-001",
					Check:    "layers",
					Severity: "error",
					File:     rel,
					Line:     fset.Position(ts.Pos()).Line,
					Found:    "type " + name + " interface",
					Why:      "Repository interfaces are not required and must not become a second architecture (ADR-0003).",
					How:      "Delete the interface. Use orm.Query[T]() or an optional concrete struct wrapper.",
					See:      archSee + " §J · ADR-0003",
				})
				return true
			}
			if _, isStruct := ts.Type.(*ast.StructType); !isStruct {
				return true
			}
			rule, why := forbiddenLayerType(name)
			if rule == "" {
				return true
			}
			how := "Use a controller, FormRequest, or named application service verb (Place, Publish). Do not add UseCase/DTO/WebService types."
			see := archSee + " §B · ADR-0001 · ADR-0009"
			if rule == "APP-REP-001" {
				how = "Delete the generic repository abstraction. Optional concrete *Repository structs wrapping ORM are allowed."
				see = archSee + " §J · ADR-0003"
			}
			out = append(out, Finding{
				Rule:     rule,
				Check:    "layers",
				Severity: "error",
				File:     rel,
				Line:     fset.Position(ts.Pos()).Line,
				Found:    "type " + name,
				Why:      why,
				How:      how,
				See:      see,
			})
			return true
		})
	})
	return out, err
}

func consumerAppPackage(rel string) bool {
	rel = filepath.ToSlash(rel)
	return strings.HasPrefix(rel, "app/") || strings.HasPrefix(rel, "domain/")
}

func forbiddenLayerType(name string) (rule, why string) {
	switch {
	case strings.HasSuffix(name, "UseCase"), strings.HasSuffix(name, "Usecase"):
		return "APP-LAY-003", "UseCase types are not a ZATRANO application layer."
	case strings.HasSuffix(name, "Interactor"):
		return "APP-LAY-003", "Interactor types are a UseCase synonym, not a ZATRANO application layer."
	case strings.HasSuffix(name, "DTO"), strings.HasSuffix(name, "Dto"):
		return "APP-LAY-003", "DTO types are not a ZATRANO application layer; use FormRequest."
	case name == "WebService" || name == "ApiService" || strings.HasSuffix(name, "WebService") || strings.HasSuffix(name, "ApiService"):
		return "APP-LAY-003", "Transport-specific services are forbidden (ADR-0009). Share one optional service, two controllers."
	case strings.HasSuffix(name, "WebUseCase") || strings.HasSuffix(name, "ApiUseCase"):
		return "APP-LAY-003", "WebUseCase/ApiUseCase are forbidden (ADR-0009)."
	case name == "BaseRepository" || name == "GenericRepository" || strings.HasSuffix(name, "RepositoryFactory"):
		return "APP-REP-001", "Generic repository abstractions are not ZATRANO architecture (ADR-0003). Optional concrete *Repository structs wrapping ORM are allowed."
	default:
		return "", ""
	}
}
