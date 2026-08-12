package pulse

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/observability"
	"github.com/zatrano/framework/packages/routing"
)

// Dashboard renders a lightweight metrics pulse UI.
type Dashboard struct {
	metrics   *observability.Metrics
	startedAt time.Time
	extra     func() map[string]any
}

// New creates a pulse dashboard.
func New(metrics *observability.Metrics) *Dashboard {
	return &Dashboard{metrics: metrics, startedAt: time.Now().UTC()}
}

// WithExtra adds extra snapshot fields (queue depth, etc).
func (d *Dashboard) WithExtra(fn func() map[string]any) *Dashboard {
	d.extra = fn
	return d
}

// Snapshot merges metrics with uptime and optional extras.
func (d *Dashboard) Snapshot() map[string]any {
	snap := map[string]any{}
	if d.metrics != nil {
		for k, v := range d.metrics.Snapshot() {
			snap[k] = v
		}
	}
	snap["uptime"] = time.Since(d.startedAt).Round(time.Second).String()
	snap["started_at"] = d.startedAt.Format(time.RFC3339)
	if d.extra != nil {
		for k, v := range d.extra() {
			snap[k] = v
		}
	}
	return snap
}

// Handler serves JSON snapshot.
func (d *Dashboard) Handler() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		return http.JSON(d.Snapshot())
	}
}

// Page serves an HTML dashboard.
func (d *Dashboard) Page() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		snap := d.Snapshot()
		var b strings.Builder
		b.WriteString(`<!doctype html><html><head><meta charset="utf-8"><title>Pulse</title>
<meta http-equiv="refresh" content="5">
<style>
body{font-family:ui-sans-serif,system-ui;background:#0b1220;color:#e8eef8;margin:0;padding:2rem}
h1{color:#3dd6c6;margin:0 0 1rem}
.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:1rem;margin:1.5rem 0}
.card{background:#121a2b;border:1px solid #243044;border-radius:12px;padding:1rem}
.card .label{opacity:.7;font-size:.85rem}.card .value{font-size:1.6rem;font-weight:700;margin-top:.35rem}
table{width:100%;border-collapse:collapse}td,th{padding:.5rem;border-bottom:1px solid #243044;text-align:left}
a{color:#3dd6c6}
</style></head><body>`)
		b.WriteString(`<h1>Pulse</h1><p style="opacity:.8">Live application metrics · auto-refresh 5s · <a href="/api/pulse">JSON</a></p>`)
		b.WriteString(`<div class="grid">`)
		writeCard(&b, "Requests", fmt.Sprint(snap["requests"]))
		writeCard(&b, "Errors", fmt.Sprint(snap["errors"]))
		writeCard(&b, "Avg ms", fmt.Sprintf("%.2f", asFloat(snap["avg_ms"])))
		writeCard(&b, "Uptime", fmt.Sprint(snap["uptime"]))
		b.WriteString(`</div>`)
		b.WriteString(`<p>Slowest: <code>`)
		b.WriteString(html.EscapeString(fmt.Sprint(snap["slowest"])))
		b.WriteString(`</code> (`)
		b.WriteString(fmt.Sprintf("%.2f", asFloat(snap["slowest_ms"])))
		b.WriteString(` ms)</p>`)
		if statuses, ok := snap["status_counts"].(map[string]int64); ok && len(statuses) > 0 {
			b.WriteString(`<h2>Status codes</h2><table><thead><tr><th>Status</th><th>Count</th></tr></thead><tbody>`)
			for code, count := range statuses {
				b.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%d</td></tr>", html.EscapeString(code), count))
			}
			b.WriteString(`</tbody></table>`)
		}
		b.WriteString(`</body></html>`)
		return http.HTML(b.String())
	}
}

func writeCard(b *strings.Builder, label, value string) {
	b.WriteString(`<div class="card"><div class="label">`)
	b.WriteString(html.EscapeString(label))
	b.WriteString(`</div><div class="value">`)
	b.WriteString(html.EscapeString(value))
	b.WriteString(`</div></div>`)
}

func asFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}
