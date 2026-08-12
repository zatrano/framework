package geo_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/packages/geo"
)

func TestGeoLookup(t *testing.T) {
	r := geo.New()
	loc := r.Lookup("8.8.8.8")
	if loc.CountryCode != "US" {
		t.Fatalf("%+v", loc)
	}
	loop := r.Lookup("127.0.0.1")
	if loop.CountryCode != "LO" {
		t.Fatalf("%+v", loop)
	}
	priv := r.Lookup("10.0.0.5")
	if priv.CountryCode != "PR" {
		t.Fatalf("%+v", priv)
	}
	if priv.Source != "private" {
		t.Fatalf("source=%s", priv.Source)
	}
}

func TestGeoPutManual(t *testing.T) {
	r := geo.New()
	r.Put("203.0.113.10", geo.Location{
		Country:     "Exampleland",
		CountryCode: "EX",
		City:        "Demo City",
	})
	loc := r.Lookup("203.0.113.10")
	if loc.CountryCode != "EX" || loc.Source != "manual" {
		t.Fatalf("%+v", loc)
	}
}

func TestGeoRemoteLookupAndCache(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","country":"Germany","countryCode":"DE","city":"Berlin","lat":52.52,"lon":13.405,"query":"203.0.113.50"}`))
	}))
	defer srv.Close()

	r := geo.New()
	r.SetHTTPClient(srv.Client())
	r.SetLookupBase(srv.URL)

	loc := r.Lookup("203.0.113.50")
	if loc.CountryCode != "DE" || loc.Source != "ip-api" || loc.City != "Berlin" {
		t.Fatalf("%+v", loc)
	}
	_ = r.Lookup("203.0.113.50")
	if hits != 1 {
		t.Fatalf("expected 1 remote hit after cache, got %d", hits)
	}
}

func TestGeoRemoteFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer srv.Close()

	r := geo.New()
	r.SetHTTPClient(srv.Client())
	r.SetLookupBase(srv.URL)

	loc := r.Lookup("198.51.100.1")
	if loc.Source != "error" || loc.CountryCode != "XX" {
		t.Fatalf("%+v", loc)
	}
}
