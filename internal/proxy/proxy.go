package proxy

import (
	"bytes"
	"crypto/subtle"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/niellevince/go-proxy/internal/store"
)

const headerKey = "X-PROXY-KEY"

type Handler struct {
	path      string
	Transport http.RoundTripper

	mu      sync.Mutex
	file    *store.File
	modTime time.Time
	byHost  map[string]store.Route
}

func New(path string) *Handler {
	return &Handler{path: path}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	from := r.Host
	to := "-"
	rawReq := dumpRequest(r)
	rec := &captureWriter{ResponseWriter: w}
	var proxyErr error
	defer func() {
		log.Printf("From: %s\nTo: %s\n%s\n%s", from, to, rawReq, rec.dump(proxyErr))
	}()

	file, err := h.current()
	if err != nil {
		proxyErr = err
		http.Error(rec, "Bad Gateway", http.StatusBadGateway)
		return
	}

	host := NormalizeHost(r.Host)
	if file.HealthHost != "" && host == NormalizeHost(file.HealthHost) {
		rec.Header().Set("Content-Type", "text/plain; charset=utf-8")
		rec.WriteHeader(http.StatusOK)
		_, _ = rec.Write([]byte("ok\n"))
		return
	}

	route, ok := h.byHost[host]
	if !ok {
		http.Error(rec, "Not Found", http.StatusNotFound)
		return
	}
	from = route.From
	to = route.To
	if strings.TrimSpace(route.Key) != "" && !keyMatch(r.Header.Get(headerKey), route.Key) {
		http.Error(rec, "Forbidden", http.StatusForbidden)
		return
	}

	target, err := url.Parse(route.To)
	if err != nil || target.Host == "" || target.Scheme != "https" {
		http.Error(rec, "Bad Gateway", http.StatusBadGateway)
		return
	}

	rp := &httputil.ReverseProxy{
		Transport: h.Transport,
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.Out.Header.Del(headerKey)
		},
		FlushInterval: -1,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			proxyErr = err
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		},
	}
	rp.ServeHTTP(rec, r)
}

func dumpRequest(r *http.Request) string {
	raw, err := httputil.DumpRequest(r, true)
	if err != nil {
		return fmt.Sprintf("request dump error: %v", err)
	}
	var b strings.Builder
	for line := range strings.SplitSeq(string(raw), "\n") {
		if strings.HasPrefix(strings.ToLower(line), "x-proxy-key:") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimSuffix(b.String(), "\n")
}

type captureWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (c *captureWriter) WriteHeader(code int) {
	if c.status == 0 {
		c.status = code
		c.ResponseWriter.WriteHeader(code)
	}
}

func (c *captureWriter) Write(p []byte) (int, error) {
	if c.status == 0 {
		c.WriteHeader(http.StatusOK)
	}
	c.body.Write(p)
	return c.ResponseWriter.Write(p)
}

func (c *captureWriter) Flush() {
	if f, ok := c.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (c *captureWriter) dump(proxyErr error) string {
	status := c.status
	if status == 0 {
		status = http.StatusOK
	}
	var b strings.Builder
	fmt.Fprintf(&b, "HTTP %d\n", status)
	for name, values := range c.Header() {
		for _, value := range values {
			fmt.Fprintf(&b, "%s: %s\n", name, value)
		}
	}
	if proxyErr != nil {
		fmt.Fprintf(&b, "\nproxy error: %v\n", proxyErr)
	}
	if c.body.Len() > 0 {
		b.WriteByte('\n')
		b.Write(c.body.Bytes())
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func (h *Handler) current() (*store.File, error) {
	info, err := os.Stat(h.path)
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.file != nil && info.ModTime().Equal(h.modTime) {
		return h.file, nil
	}

	file, err := store.Load(h.path)
	if err != nil {
		if h.file != nil {
			return h.file, nil
		}
		return nil, err
	}

	byHost := make(map[string]store.Route, len(file.Proxies))
	for _, route := range file.Proxies {
		host := NormalizeHost(route.From)
		if host == "" {
			continue
		}
		if _, exists := byHost[host]; exists {
			continue
		}
		byHost[host] = route
	}

	h.file = file
	h.modTime = info.ModTime()
	h.byHost = byHost
	return h.file, nil
}

func NormalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	return strings.ToLower(host)
}

func NormalizeTarget(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("--to is required")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid --to: %w", err)
	}
	if u.Scheme != "https" || u.Host == "" {
		return "", fmt.Errorf("--to must be an https origin or a hostname")
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func keyMatch(got, want string) bool {
	if want == "" || len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
