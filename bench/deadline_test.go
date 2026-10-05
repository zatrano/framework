package bench

import (
	"bytes"
	"fmt"
	"testing"
)

func TestDeadlineCallsPerRequest(t *testing.T) {
	const n = 32
	cases := []struct {
		name  string
		serve serveFunc
	}{
		{"old", serveZatranoTier0},
		{"run-v301", serveZatranoTier0RunV301},
		{"run-head", serveZatranoTier0Run},
		{"tier1-old", serveZatranoTier1},
		{"tier1-run-head", serveZatranoTier1Run},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := bytes.Repeat(request, n)
			buf := &bytes.Buffer{}
			c := &countConn{memConn: &memConn{r: bytes.NewReader(payload), w: buf}}
			err := tc.serve(c)
			if buf.Len() == 0 {
				t.Fatalf("no response: %v", err)
			}
			t.Logf("SetReadDeadline=%d SetWriteDeadline=%d over %d requests (%.2f read/req %.2f write/req)",
				c.readDL, c.writeDL, n, float64(c.readDL)/n, float64(c.writeDL)/n)
			fmt.Printf("DEADLINE %s read=%d write=%d n=%d\n", tc.name, c.readDL, c.writeDL, n)
		})
	}
}
