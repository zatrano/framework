package cookie

import (
	stdhttp "net/http"
	"strings"
	"sync"
	"time"

	"github.com/zatrano/framework/v2/kernel/env"
)

// QueuedCookie is a cookie waiting to be attached to a response.
type QueuedCookie struct {
	Name     string
	Value    string
	Minutes  int
	Path     string
	Domain   string
	Secure   bool
	HTTPOnly bool
	SameSite stdhttp.SameSite
	Raw      *stdhttp.Cookie
}

// Jar queues cookies for the current response cycle.
type Jar struct {
	queued       []*QueuedCookie
	queuedForget []string
}

// NewJar creates a cookie jar.
func NewJar() *Jar {
	return &Jar{
		queued:       make([]*QueuedCookie, 0),
		queuedForget: make([]string, 0),
	}
}

// Queue queues a cookie.
func (j *Jar) Queue(name, value string, minutes int) *Jar {
	j.removeForget(name)
	j.queued = append(j.queued, &QueuedCookie{
		Name:     name,
		Value:    value,
		Minutes:  minutes,
		Path:     "/",
		Secure:   SecureByDefault(),
		HTTPOnly: true,
		SameSite: stdhttp.SameSiteLaxMode,
	})
	return j
}

// Forever queues a long-lived cookie (~5 years).
func (j *Jar) Forever(name, value string) *Jar {
	return j.Queue(name, value, 60*24*365*5)
}

// ForeverSecure queues a long-lived cookie with Secure set when secure is true
// or when production / COOKIE_SECURE / SESSION_SECURE defaults apply.
func (j *Jar) ForeverSecure(name, value string, secure bool) *Jar {
	j.removeForget(name)
	j.queued = append(j.queued, &QueuedCookie{
		Name:     name,
		Value:    value,
		Minutes:  60 * 24 * 365 * 5,
		Path:     "/",
		Secure:   secure || SecureByDefault(),
		HTTPOnly: true,
		SameSite: stdhttp.SameSiteLaxMode,
	})
	return j
}

// Forget queues a cookie deletion.
func (j *Jar) Forget(name string) *Jar {
	j.removeQueued(name)
	j.queuedForget = append(j.queuedForget, name)
	return j
}

// Make builds a standard cookie.
func Make(name, value string, minutes int) *stdhttp.Cookie {
	cookie := &stdhttp.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   SecureByDefault(),
		SameSite: stdhttp.SameSiteLaxMode,
	}
	if minutes > 0 {
		cookie.MaxAge = minutes * 60
		cookie.Expires = time.Now().Add(time.Duration(minutes) * time.Minute)
	} else if minutes < 0 {
		cookie.MaxAge = -1
		cookie.Expires = time.Unix(0, 0)
	}
	return cookie
}

// SecureByDefault reports whether framework cookie helpers should set Secure.
// Production is the bootstrapped application policy (SetProductionPolicy),
// not a live APP_ENV parse. COOKIE_SECURE / SESSION_SECURE still force
// Secure in any environment. Explicit cookies (QueueRaw, Response.Cookie,
// WithCookieOptions) are not implied by this.
func SecureByDefault() bool {
	if defaultSecure() {
		return true
	}
	prodMu.RLock()
	on := prodOn
	prodMu.RUnlock()
	return on
}

func defaultSecure() bool {
	if strings.EqualFold(strings.TrimSpace(env.Get("COOKIE_SECURE", "")), "true") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(env.Get("SESSION_SECURE", "")), "true")
}

var (
	prodMu sync.RWMutex
	prodOn bool
)

// SetProductionPolicy records the bootstrapped IsProduction value for
// framework cookie helpers. Call once during Bootstrap.
func SetProductionPolicy(on bool) {
	prodMu.Lock()
	prodOn = on
	prodMu.Unlock()
}

// ForeverCookie builds a long-lived cookie.
func ForeverCookie(name, value string) *stdhttp.Cookie {
	return Make(name, value, 60*24*365*5)
}

// ForgetCookie builds an expired cookie.
func ForgetCookie(name string) *stdhttp.Cookie {
	return Make(name, "", -1)
}

// Get reads a cookie value from the request.
func Get(r *stdhttp.Request, name string, fallback ...string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	return cookie.Value
}

// Has reports whether a cookie exists.
func Has(r *stdhttp.Request, name string) bool {
	_, err := r.Cookie(name)
	return err == nil
}

// Apply writes queued cookies onto a response writer helper list.
func (j *Jar) Apply() []*stdhttp.Cookie {
	out := make([]*stdhttp.Cookie, 0, len(j.queued)+len(j.queuedForget))
	for _, item := range j.queued {
		if item.Raw != nil {
			out = append(out, item.Raw)
			continue
		}
		c := Make(item.Name, item.Value, item.Minutes)
		if item.Path != "" {
			c.Path = item.Path
		}
		c.Domain = item.Domain
		c.Secure = item.Secure
		c.HttpOnly = item.HTTPOnly
		c.SameSite = item.SameSite
		out = append(out, c)
	}
	for _, name := range j.queuedForget {
		out = append(out, ForgetCookie(name))
	}
	return out
}

// Clear clears the queue.
func (j *Jar) Clear() {
	j.queued = j.queued[:0]
	j.queuedForget = j.queuedForget[:0]
}

// QueueRaw queues a fully built cookie.
func (j *Jar) QueueRaw(cookie *stdhttp.Cookie) *Jar {
	if cookie != nil {
		j.removeForget(cookie.Name)
	}
	j.queued = append(j.queued, &QueuedCookie{Raw: cookie})
	return j
}

func (j *Jar) removeForget(name string) {
	if j == nil || len(j.queuedForget) == 0 {
		return
	}
	out := j.queuedForget[:0]
	for _, n := range j.queuedForget {
		if n != name {
			out = append(out, n)
		}
	}
	j.queuedForget = out
}

func (j *Jar) removeQueued(name string) {
	if j == nil || len(j.queued) == 0 {
		return
	}
	out := j.queued[:0]
	for _, item := range j.queued {
		n := item.Name
		if item.Raw != nil {
			n = item.Raw.Name
		}
		if n != name {
			out = append(out, item)
		}
	}
	j.queued = out
}
