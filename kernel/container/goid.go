package container

import (
	"bytes"
	"runtime"
	"strconv"
)

func goroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	s := buf[:n]
	if i := bytes.IndexByte(s, ' '); i >= 0 {
		s = s[i+1:]
		if j := bytes.IndexByte(s, ' '); j >= 0 {
			id, _ := strconv.ParseUint(string(s[:j]), 10, 64)
			return id
		}
	}
	return 0
}
