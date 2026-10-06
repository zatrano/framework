package doctor

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zatrano/framework/v3/core/kernel/http"
)

func checkBodyBudget(root string) ([]Finding, error) {
	budget := inflightBudget(root)
	if budget < 0 {
		return nil, nil
	}
	var out []Finding
	err := walkConsumerGo(root, func(rel, _ string, fset *token.FileSet, file *ast.File) {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "BodyLimit" {
				return true
			}
			limit, ok := constInt(call.Args[0])
			if !ok || limit <= budget {
				return true
			}
			pos := fset.Position(call.Pos())
			out = append(out, Finding{
				Rule:     "APP-HTTP-006",
				Check:    "body-budget",
				Severity: "error",
				File:     rel,
				Line:     pos.Line,
				Found:    "BodyLimit above HTTP_MAX_INFLIGHT_BODY_BYTES",
				Why:      "A route cap larger than the in-flight body budget can never be reserved. The request is rejected with 413, and production boot fails.",
				How:      "Lower BodyLimit, or raise HTTP_MAX_INFLIGHT_BODY_BYTES. HTTP_STRICT_LIMITS fails boot outside production too.",
			})
			return true
		})
	})
	return out, err
}

func inflightBudget(root string) int64 {
	budget := int64(http.DefaultMaxInflightBodyBytes)
	raw, err := os.ReadFile(filepath.Join(root, ".env"))
	if err != nil {
		return budget
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "HTTP_MAX_INFLIGHT_BODY_BYTES" {
			continue
		}
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return budget
		}
		return n
	}
	return budget
}

func constInt(e ast.Expr) (int64, bool) {
	switch n := e.(type) {
	case *ast.BasicLit:
		if n.Kind != token.INT {
			return 0, false
		}
		v, err := strconv.ParseInt(n.Value, 0, 64)
		return v, err == nil
	case *ast.ParenExpr:
		return constInt(n.X)
	case *ast.BinaryExpr:
		if n.Op != token.SHL && n.Op != token.MUL {
			return 0, false
		}
		x, ok1 := constInt(n.X)
		y, ok2 := constInt(n.Y)
		if !ok1 || !ok2 {
			return 0, false
		}
		if n.Op == token.MUL {
			return x * y, true
		}
		if y < 0 || y >= 63 {
			return 0, false
		}
		return x << y, true
	default:
		return 0, false
	}
}
