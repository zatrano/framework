package geo

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Location is a coarse geo result.
type Location struct {
	IP          string  `json:"ip"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city,omitempty"`
	Lat         float64 `json:"lat,omitempty"`
	Lon         float64 `json:"lon,omitempty"`
	Source      string  `json:"source"`
}

// Resolver looks up IP geolocation via exact maps, private-range rules, and ip-api.com.
type Resolver struct {
	mu     sync.RWMutex
	exact  map[string]Location
	client *http.Client
	base   string // override for tests; empty uses ip-api.com
}

// New creates a geo resolver with a few demo mappings.
func New() *Resolver {
	r := &Resolver{
		exact: make(map[string]Location),
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
		base: "http://ip-api.com/json",
	}
	r.Put("8.8.8.8", Location{Country: "United States", CountryCode: "US", City: "Mountain View", Lat: 37.386, Lon: -122.084, Source: "manual"})
	r.Put("1.1.1.1", Location{Country: "Australia", CountryCode: "AU", City: "Sydney", Lat: -33.8688, Lon: 151.2093, Source: "manual"})
	return r
}

// SetHTTPClient replaces the HTTP client used for remote lookups (tests).
func (r *Resolver) SetHTTPClient(c *http.Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.client = c
}

// SetLookupBase overrides the ip-api JSON base URL (tests).
func (r *Resolver) SetLookupBase(base string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.base = strings.TrimRight(base, "/")
}

// Put registers an exact IP mapping.
func (r *Resolver) Put(ip string, loc Location) {
	r.mu.Lock()
	defer r.mu.Unlock()
	loc.IP = ip
	if loc.Source == "" {
		loc.Source = "manual"
	}
	r.exact[ip] = loc
}

// Lookup resolves an IP address.
func (r *Resolver) Lookup(ip string) Location {
	ip = strings.TrimSpace(ip)
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	r.mu.RLock()
	if loc, ok := r.exact[ip]; ok {
		r.mu.RUnlock()
		return loc
	}
	r.mu.RUnlock()

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return Location{IP: ip, Country: "Unknown", CountryCode: "XX", Source: "invalid"}
	}
	if parsed.IsLoopback() {
		return Location{IP: ip, Country: "Local", CountryCode: "LO", City: "Loopback", Source: "private"}
	}
	if parsed.IsPrivate() {
		return Location{IP: ip, Country: "Private Network", CountryCode: "PR", Source: "private"}
	}
	if parsed.IsLinkLocalUnicast() || parsed.IsLinkLocalMulticast() {
		return Location{IP: ip, Country: "Link Local", CountryCode: "LL", Source: "private"}
	}

	loc, ok := r.lookupRemote(ip)
	if !ok {
		return Location{IP: ip, Country: "Unknown", CountryCode: "XX", Source: "error"}
	}
	r.Put(ip, loc)
	return loc
}

type ipAPIResponse struct {
	Status      string  `json:"status"`
	Message     string  `json:"message"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	City        string  `json:"city"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Query       string  `json:"query"`
}

func (r *Resolver) lookupRemote(ip string) (Location, bool) {
	r.mu.RLock()
	client := r.client
	base := r.base
	r.mu.RUnlock()
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	if base == "" {
		base = "http://ip-api.com/json"
	}

	url := fmt.Sprintf("%s/%s?fields=status,message,country,countryCode,city,lat,lon,query", base, ip)
	resp, err := client.Get(url)
	if err != nil {
		return Location{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Location{}, false
	}

	var body ipAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Location{}, false
	}
	if !strings.EqualFold(body.Status, "success") {
		return Location{}, false
	}

	outIP := body.Query
	if outIP == "" {
		outIP = ip
	}
	return Location{
		IP:          outIP,
		Country:     body.Country,
		CountryCode: body.CountryCode,
		City:        body.City,
		Lat:         body.Lat,
		Lon:         body.Lon,
		Source:      "ip-api",
	}, true
}
