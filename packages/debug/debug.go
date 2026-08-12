package debug

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Dump formats values for debugging and returns the text.
func Dump(values ...any) string {
	var b strings.Builder
	for i, v := range values {
		if i > 0 {
			b.WriteString("\n---\n")
		}
		raw, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			b.WriteString(fmt.Sprintf("%#v", v))
			continue
		}
		b.Write(raw)
	}
	return b.String()
}

// Log writes Dump output to stderr.
func Log(values ...any) {
	fmt.Fprintln(os.Stderr, Dump(values...))
}

var exitFunc = os.Exit

// DD dumps values to stderr and exits the process (die and dump).
func DD(values ...any) {
	Log(values...)
	exitFunc(1)
}

// SetExitFunc overrides process exit (tests).
func SetExitFunc(fn func(int)) {
	if fn == nil {
		exitFunc = os.Exit
		return
	}
	exitFunc = fn
}
