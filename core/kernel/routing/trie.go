package routing

import (
	"fmt"
	"strings"
)

type segKind int

const (
	segStatic segKind = iota
	segParam
	segCatchAll
)

type routeSeg struct {
	kind     segKind
	value    string
	optional bool
}

type trieNode struct {
	static    map[string]*trieNode
	param     *trieNode
	paramName string
	catchAll  *trieNode
	catchName string
	route     *Route
}

func newTrieNode() *trieNode {
	return &trieNode{static: make(map[string]*trieNode)}
}

func parseRouteSegs(path string) []routeSeg {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	raw := strings.Split(path, "/")
	out := make([]routeSeg, 0, len(raw))
	for _, part := range raw {
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			name := strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
			optional := strings.HasSuffix(name, "?")
			name = strings.TrimSuffix(name, "?")
			catchAll := strings.HasPrefix(name, "*") || strings.HasSuffix(name, "*")
			name = strings.TrimPrefix(name, "*")
			name = strings.TrimSuffix(name, "*")
			kind := segParam
			if catchAll {
				kind = segCatchAll
				optional = false
			}
			out = append(out, routeSeg{kind: kind, value: name, optional: optional})
			continue
		}
		out = append(out, routeSeg{kind: segStatic, value: part})
	}
	return out
}

// requestSegsBuf splits path on '/' into buf when it fits.
// The segment strings alias path. Behavior matches strings.Split after trim.
func requestSegsBuf(path string, buf []string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	n := 1
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			n++
		}
	}
	var segs []string
	if n <= len(buf) {
		segs = buf[:n]
	} else {
		segs = make([]string, n)
	}
	start := 0
	k := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			segs[k] = path[start:i]
			k++
			start = i + 1
		}
	}
	segs[k] = path[start:]
	return segs
}

func (n *trieNode) insert(segs []routeSeg, route *Route) error {
	if len(segs) == 0 {
		if n.route != nil {
			return ambiguousRoute(n.route, route)
		}
		n.route = route
		return nil
	}
	s := segs[0]
	rest := segs[1:]
	switch s.kind {
	case segStatic:
		child := n.static[s.value]
		if child == nil {
			child = newTrieNode()
			n.static[s.value] = child
		}
		return child.insert(rest, route)
	case segParam:
		if n.param != nil && n.paramName != "" && n.paramName != s.value {
			return fmt.Errorf("router: ambiguous route %s %s conflicts with parameter {%s}", route.Method, route.Path, n.paramName)
		}
		if n.param == nil {
			n.param = newTrieNode()
			n.paramName = s.value
		}
		if s.optional && len(rest) == 0 {
			if n.route != nil && n.route != route {
				return ambiguousRoute(n.route, route)
			}
			n.route = route
		}
		return n.param.insert(rest, route)
	case segCatchAll:
		if n.catchAll != nil && n.catchName != "" && n.catchName != s.value {
			return fmt.Errorf("router: ambiguous route %s %s conflicts with catch-all {%s}", route.Method, route.Path, n.catchName)
		}
		if n.catchAll == nil {
			n.catchAll = newTrieNode()
			n.catchName = s.value
		}
		if n.catchAll.route != nil {
			return ambiguousRoute(n.catchAll.route, route)
		}
		n.catchAll.route = route
		return nil
	}
	return nil
}

func ambiguousRoute(existing, incoming *Route) error {
	return fmt.Errorf("router: ambiguous route %s %s conflicts with %s %s", incoming.Method, incoming.Path, existing.Method, existing.Path)
}

func (n *trieNode) lookup(parts []string, hit *routeHit) *Route {
	if n == nil {
		return nil
	}
	if len(parts) == 0 {
		return n.route
	}
	if child := n.static[parts[0]]; child != nil {
		if route := child.lookup(parts[1:], hit); route != nil {
			return route
		}
	}
	if n.param != nil {
		hit.pushParam(n.paramName, parts[0])
		if route := n.param.lookup(parts[1:], hit); route != nil {
			return route
		}
		hit.popParam(n.paramName)
	}
	if n.catchAll != nil && n.catchAll.route != nil {
		hit.pushParam(n.catchName, strings.Join(parts, "/"))
		return n.catchAll.route
	}
	return nil
}
