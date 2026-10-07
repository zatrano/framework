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

func checkTrustedProxyShare(root string) ([]Finding, error) {
	if perClientShareDisabled(root) {
		return nil, nil
	}
	if strings.TrimSpace(envValue(root, "TRUSTED_PROXIES")) != "" {
		return nil, nil
	}
	return []Finding{{
		Rule:     "APP-HTTP-007",
		Check:    "trusted-proxy-share",
		Severity: "warning",
		Found:    "TRUSTED_PROXIES is unset",
		Why:      "The per-client in-flight body share applies only when clients can be told apart. A loopback, private, link-local, CGNAT, or unique-local peer with no trusted proxy does not get a share; only the global budget applies. Behind a reverse proxy every client is that one address until TRUSTED_PROXIES is set.",
		How:      "Set TRUSTED_PROXIES to the proxy addresses. A negative HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT disables the share.",
	}}, nil
}

func perClientShareDisabled(root string) bool {
	raw := envValue(root, "HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT")
	if strings.TrimSpace(raw) == "" {
		raw = envValue(root, "HTTP_MAX_INFLIGHT_BODY_BYTES")
	}
	if strings.TrimSpace(raw) == "" {
		return false
	}
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	return err == nil && n < 0
}

func envValue(root, want string) string {
	raw, err := os.ReadFile(filepath.Join(root, ".env"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != want {
			continue
		}
		return strings.Trim(strings.TrimSpace(val), `"'`)
	}
	return ""
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
