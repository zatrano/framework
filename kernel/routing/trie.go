package routing

import "strings"

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

func requestSegs(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func (n *trieNode) insert(segs []routeSeg, route *Route) {
	if len(segs) == 0 {
		if n.route == nil {
			n.route = route
		}
		return
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
		child.insert(rest, route)
	case segParam:
		if n.param == nil {
			n.param = newTrieNode()
			n.paramName = s.value
		}
		if s.optional && len(rest) == 0 && n.route == nil {
			n.route = route
		}
		n.param.insert(rest, route)
	case segCatchAll:
		if n.catchAll == nil {
			n.catchAll = newTrieNode()
			n.catchName = s.value
		}
		if n.catchAll.route == nil {
			n.catchAll.route = route
		}
	}
}

func (n *trieNode) lookup(parts []string, params map[string]string) *Route {
	if n == nil {
		return nil
	}
	if len(parts) == 0 {
		return n.route
	}
	if child := n.static[parts[0]]; child != nil {
		if route := child.lookup(parts[1:], params); route != nil {
			return route
		}
	}
	if n.param != nil {
		params[n.paramName] = parts[0]
		if route := n.param.lookup(parts[1:], params); route != nil {
			return route
		}
		delete(params, n.paramName)
	}
	if n.catchAll != nil && n.catchAll.route != nil {
		params[n.catchName] = strings.Join(parts, "/")
		return n.catchAll.route
	}
	return nil
}
