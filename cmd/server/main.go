package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/niellevince/go-proxy/internal/proxy"
	"github.com/niellevince/go-proxy/internal/store"
)

func main() {
	loadEnv(".env")
	port := portFromEnv()

	path := store.DefaultPath
	if _, err := os.Stat(path); err != nil {
		log.Fatalf("proxies config: %v", err)
	}

	addr := fmt.Sprintf(":%d", port)
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, proxy.New(path)); err != nil {
		log.Fatal(err)
	}
}

func portFromEnv() int {
	raw := strings.TrimSpace(os.Getenv("PORT"))
	if raw == "" {
		return 8000
	}
	port, err := strconv.Atoi(raw)
	if err != nil || port < 1 || port > 65535 {
		return 8000
	}
	return port
}

func loadEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		_ = os.Setenv(key, val)
	}
}
