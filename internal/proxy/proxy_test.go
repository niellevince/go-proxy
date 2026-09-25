package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/niellevince/go-proxy/internal/store"
)

func TestRouting(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != r.URL.Host && r.Header.Get("X-PROXY-KEY") != "" {
			t.Errorf("key forwarded: %s", r.Header.Get("X-PROXY-KEY"))
		}
		if r.Header.Get("X-PROXY-KEY") != "" {
			http.Error(w, "key leaked", http.StatusBadRequest)
			return
		}
		w.Header().Set("X-Upstream-Host", r.Host)
		_, _ = io.WriteString(w, r.URL.RequestURI())
	}))
	defer upstream.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "proxies.json")
	if err := store.Save(path, &store.File{
		HealthHost: "health.example.com",
		Proxies: []store.Route{
			{
				From: "hello.world.com",
				To:   upstream.URL,
				Key:  "secret",
			},
			{
				From: "public.example.com",
				To:   upstream.URL,
				Key:  "",
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	h := New(path)
	h.Transport = upstream.Client().Transport

	req := httptest.NewRequest(http.MethodGet, "http://hello.world.com/a?b=1", nil)
	req.Host = "Hello.World.com:8000"
	req.Header.Set("X-PROXY-KEY", "secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("proxy status %d body %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "/a?b=1" {
		t.Fatalf("body %q", rec.Body.String())
	}
	if got := rec.Header().Get("X-Upstream-Host"); got != upstream.Listener.Addr().String() {
		t.Fatalf("upstream host %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "http://hello.world.com/", nil)
	req.Host = "hello.world.com"
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("missing key status %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "http://public.example.com/open", nil)
	req.Host = "public.example.com"
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "/open" {
		t.Fatalf("public proxy %d %q", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "http://other.example/", nil)
	req.Host = "other.example"
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown status %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "http://health.example.com/", nil)
	req.Host = "health.example.com"
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "ok\n" {
		t.Fatalf("health %d %q", rec.Code, rec.Body.String())
	}
}

func TestReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proxies.json")
	if err := store.Save(path, store.Empty()); err != nil {
		t.Fatal(err)
	}
	h := New(path)

	req := httptest.NewRequest(http.MethodGet, "http://hello.world.com/", nil)
	req.Host = "hello.world.com"
	req.Header.Set("X-PROXY-KEY", "secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("before add %d", rec.Code)
	}

	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	time.Sleep(20 * time.Millisecond)
	if err := store.Save(path, &store.File{
		Proxies: []store.Route{{
			From: "hello.world.com",
			To:   upstream.URL,
			Key:  "secret",
		}},
	}); err != nil {
		t.Fatal(err)
	}

	h.Transport = upstream.Client().Transport
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("after reload %d %s", rec.Code, rec.Body.String())
	}
}

func TestNormalizeTarget(t *testing.T) {
	got, err := NormalizeTarget("hi.world.com")
	if err != nil || got != "https://hi.world.com" {
		t.Fatalf("bare: %q %v", got, err)
	}
	got, err = NormalizeTarget("https://hi.world.com/base")
	if err != nil || got != "https://hi.world.com/base" {
		t.Fatalf("url: %q %v", got, err)
	}
	if _, err := NormalizeTarget("http://hi.world.com"); err == nil {
		t.Fatal("http should fail")
	}
}
