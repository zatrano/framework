package acquire

import (
	"fmt"
	"strings"
)

// DryRunReport is Contract A: what Execute would invoke, without invoking it.
// It is not a mutation result and not enablement. Fields must not claim
// installed / acquired / applied.
type DryRunReport struct {
	Root     string
	Module   string
	Selected string
	GoGetArg string
	Command  []string
}

// DryRun reports the concrete acquisition command for req. It does not call
// Execute, Invoke, lockMutation, recovery, or a Runner. It does not write
// go.mod / go.sum. latest is rejected; GoGetArg is not reinterpreted.
func DryRun(req Request) (DryRunReport, error) {
	root := strings.TrimSpace(req.Root)
	if root == "" {
		return DryRunReport{}, fmt.Errorf("acquire: module root required")
	}
	arg := strings.TrimSpace(req.GoGetArg)
	if arg == "" {
		return DryRunReport{}, fmt.Errorf("acquire: empty GoGetArg")
	}
	if strings.EqualFold(arg, "latest") || strings.HasSuffix(strings.ToLower(arg), "@latest") {
		return DryRunReport{}, fmt.Errorf("acquire: latest must not reach Apply")
	}
	goBin := strings.TrimSpace(req.Go)
	if goBin == "" {
		goBin = "go"
	}
	module, selected := splitConcreteArg(arg)
	return DryRunReport{
		Root:     root,
		Module:   module,
		Selected: selected,
		GoGetArg: arg,
		Command:  []string{goBin, "get", arg},
	}, nil
}

// DryRunTargets reports each concrete GoGetArg. It does not execute, lock,
// or restore files. The first invalid argument fails the call.
func DryRunTargets(root string, args []string) ([]DryRunReport, error) {
	out := make([]DryRunReport, 0, len(args))
	for _, arg := range args {
		rep, err := DryRun(Request{Root: root, GoGetArg: arg})
		if err != nil {
			return nil, err
		}
		out = append(out, rep)
	}
	return out, nil
}

func splitConcreteArg(arg string) (module, selected string) {
	i := strings.LastIndex(arg, "@")
	if i <= 0 || i == len(arg)-1 {
		return arg, ""
	}
	return arg[:i], arg[i+1:]
}
