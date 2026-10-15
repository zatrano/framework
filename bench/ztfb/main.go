// Command ztfb compares a production ZATRANO process with Fiber v3.6.0.
//
//	ztfb serve zatrano 127.0.0.1:18181
//	ztfb serve fiber 127.0.0.1:18181
//	ztfb run -o ../results/baseline.jsonl
//
// Both servers are this binary. The load generator is a separate process from
// the server. On a multi-core machine the default affinity pins the server to
// logical CPU 0 (GOMAXPROCS=1) and the client to the remaining CPUs, which is
// the -cpu split. -affinity shared leaves both on every CPU.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = cmdServe(os.Args[2:])
	case "run":
		err = cmdRun(os.Args[2:])
	case "soak":
		err = cmdSoak(os.Args[2:])
	case "load":
		err = cmdLoad(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: ztfb serve <zatrano|fiber|fiber-tuned|rawhttp|fasthttp> <addr>")
	fmt.Fprintln(os.Stderr, "       ztfb run [-o file] [-affinity split|shared] [-rounds N] [-client go|wrk] [-config default|pooled]")
	fmt.Fprintln(os.Stderr, "       ztfb soak [-o file] [-affinity split|shared]")
	fmt.Fprintln(os.Stderr, "       ztfb load <addr> <route> <conns> <seconds>")
}
