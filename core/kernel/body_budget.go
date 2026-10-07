package kernel

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/zatrano/framework/v3/core/kernel/env"
	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

const clientBudgetShards = 32

func perClientBodyCap(total int64) int64 {
	if total < 0 {
		return -1
	}
	raw, ok := env.Lookup("HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT")
	if !ok || strings.TrimSpace(raw) == "" {
		return total / 4
	}
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return total / 4
	}
	return n
}

func clientBudgetKey(ip string) string {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		if ip == "" {
			return "unknown"
		}
		return ip
	}
	if v4 := parsed.To4(); v4 != nil {
		return v4.String()
	}
	masked := parsed.Mask(net.CIDRMask(64, 128))
	return masked.String()
}

// clientShareKey is the per-client budget key when peers can be told apart.
// A trusted proxy uses the resolved client address. Otherwise only a global
// unicast address that is not private, link-local, loopback, or CGNAT is a key.
// The bool is false when the share must not apply.
func clientShareKey(peer, client string, trusted bool) (string, bool) {
	peer = remoteHostString(peer)
	client = strings.TrimSpace(client)
	if trusted {
		if client == "" || client == peer {
			return "", false
		}
		return clientBudgetKey(client), true
	}
	if !distinctClientAddr(peer) {
		return "", false
	}
	return clientBudgetKey(peer), true
}

func remoteHostString(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

func distinctClientAddr(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || isCGNAT(ip) {
		return false
	}
	return true
}

func isCGNAT(ip net.IP) bool {
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
	return v4[0] == 100 && v4[1]&0xc0 == 64
}

func (app *Application) noteUndistinguishedClient() {
	if app == nil || app.logger == nil {
		return
	}
	app.shareSkipOnce.Do(func() {
		app.logger.Warningf("per-client body share is off for this peer because it is not a distinct client; set TRUSTED_PROXIES when the process sits behind a reverse proxy")
	})
}

func clientShard(key string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}
	return h % clientBudgetShards
}

func (app *Application) rejectOversizedBodyLimits() error {
	if app == nil || app.router == nil {
		return nil
	}
	lim := app.ensureBodyLimits()
	over := app.router.BodyLimitsAbove(lim.inflight)
	if len(over) == 0 {
		return nil
	}
	msg := "route body limit exceeds HTTP_MAX_INFLIGHT_BODY_BYTES: " + strings.Join(over, "; ")
	if app.logger != nil {
		app.logger.Warningf("%s", msg)
	}
	if app.IsProduction() || env.GetBool("HTTP_STRICT_LIMITS", false) {
		return fmt.Errorf("%s", msg)
	}
	return nil
}

// bodyBudgetState is the in-flight body reservation.
// A hold is released exactly once, from the handler defer or from
// ConnState(StateClosed), whichever runs first.
type bodyBudgetState struct {
	mu       sync.Mutex
	reserved int64
	holds    map[net.Conn]*bodyHold
	shards   [clientBudgetShards]clientBudgetShard
}

type clientBudgetShard struct {
	mu   sync.Mutex
	used map[string]int64
}

type bodyHold struct {
	n        int64
	req      uint64
	key      string
	released bool
}

// BodyReserved is the number of body bytes currently reserved.
func (app *Application) BodyReserved() int64 {
	if app == nil {
		return 0
	}
	app.bodyBudget.mu.Lock()
	defer app.bodyBudget.mu.Unlock()
	return app.bodyBudget.reserved
}

func (app *Application) grantBody(ctx *rawhttp.Ctx, lim bodyLimitSnap, chosen int64, contentLength int, contentType []byte) rawhttp.RequestConfig {
	if !lim.sustainable(chosen) {
		// One request can never fit. The engine answers 413. Nothing is reserved.
		return http.BodyLimitConfig(1)
	}
	if ctx == nil || lim.inflight < 0 {
		return http.BodyLimitConfig(chosen)
	}
	n := bodyReserveAmount(lim, contentType, contentLength, chosen)
	if n > 0 && !app.reserveBody(ctx, n) {
		return rawhttp.RequestConfig{RejectStatus: 503, RejectRetryAfter: 1}
	}
	return http.BodyLimitConfig(chosen)
}

func bodyReserveAmount(lim bodyLimitSnap, contentType []byte, contentLength int, chosen int64) int64 {
	if contentLength > 0 {
		if int64(contentLength) > chosen {
			return 0
		}
		return int64(contentLength)
	}
	if contentLength < 0 {
		worst := chosen
		if earlyBodyMediaBytes(contentType) && lim.maxBody < worst {
			worst = lim.maxBody
		}
		return worst
	}
	return 0
}

func (app *Application) reserveBody(ctx *rawhttp.Ctx, n int64) bool {
	conn := ctx.Conn()
	if conn == nil || n <= 0 {
		return true
	}
	reqN := ctx.ConnRequestNum()
	app.bodyBudget.mu.Lock()
	if h := app.bodyBudget.holds[conn]; h != nil && h.req == reqN && !h.released {
		app.bodyBudget.mu.Unlock()
		return true
	}
	app.bodyBudget.mu.Unlock()
	peer := ""
	if ra := conn.RemoteAddr(); ra != nil {
		peer = ra.String()
	}
	key, apply := clientShareKey(peer, ctx.ClientIP(), len(trustedProxiesForServer()) > 0)
	lim := app.bodyLimitSnap
	if lim.perClient >= 0 && !apply {
		app.noteUndistinguishedClient()
	}
	var shard *clientBudgetShard
	if apply {
		shard = &app.bodyBudget.shards[clientShard(key)]
	}
	app.bodyBudget.mu.Lock()
	defer app.bodyBudget.mu.Unlock()
	if shard != nil {
		shard.mu.Lock()
		defer shard.mu.Unlock()
	}
	if h := app.bodyBudget.holds[conn]; h != nil && h.req == reqN && !h.released {
		return true
	}
	lim = app.bodyLimitSnap
	if apply && lim.perClient >= 0 {
		if shard.used[key]+n > lim.perClient {
			return false
		}
	}
	if lim.inflight >= 0 && app.bodyBudget.reserved+n > lim.inflight {
		return false
	}
	if app.bodyBudget.holds == nil {
		app.bodyBudget.holds = make(map[net.Conn]*bodyHold)
	}
	holdKey := ""
	if apply && lim.perClient >= 0 {
		if shard.used == nil {
			shard.used = make(map[string]int64)
		}
		shard.used[key] += n
		holdKey = key
	}
	app.bodyBudget.reserved += n
	app.bodyBudget.holds[conn] = &bodyHold{n: n, req: reqN, key: holdKey}
	return true
}

// ClientReserved is the sum of per-client shares still held.
func (app *Application) ClientReserved() int64 {
	if app == nil {
		return 0
	}
	var n int64
	for i := range app.bodyBudget.shards {
		sh := &app.bodyBudget.shards[i]
		sh.mu.Lock()
		for _, v := range sh.used {
			n += v
		}
		sh.mu.Unlock()
	}
	return n
}

// releaseBody frees the reservation for conn. reqN 0 releases whatever hold
// is still there (connection close). A positive reqN releases only that request.
func (app *Application) releaseBody(conn net.Conn, reqN uint64) {
	if app == nil || conn == nil {
		return
	}
	app.bodyBudget.mu.Lock()
	defer app.bodyBudget.mu.Unlock()
	h := app.bodyBudget.holds[conn]
	if h == nil || h.released {
		return
	}
	if reqN != 0 && h.req != reqN {
		return
	}
	h.released = true
	app.bodyBudget.reserved -= h.n
	if app.bodyBudget.reserved < 0 {
		app.bodyBudget.reserved = 0
	}
	if h.key != "" {
		sh := &app.bodyBudget.shards[clientShard(h.key)]
		sh.mu.Lock()
		sh.used[h.key] -= h.n
		if sh.used[h.key] <= 0 {
			delete(sh.used, h.key)
		}
		sh.mu.Unlock()
	}
	delete(app.bodyBudget.holds, conn)
}
