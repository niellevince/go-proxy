package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/niellevince/go-proxy/internal/proxy"
	"github.com/niellevince/go-proxy/internal/store"
)

func main() {
	from := flag.String("from", "", "incoming hostname")
	to := flag.String("to", "", "upstream hostname or https origin")
	path := flag.String("file", store.DefaultPath, "proxies config path")
	flag.Parse()

	if err := add(*path, *from, *to); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func add(path, from, to string) error {
	host := proxy.NormalizeHost(from)
	if host == "" || strings.Contains(host, "/") {
		return fmt.Errorf("--from must be a hostname")
	}
	target, err := proxy.NormalizeTarget(to)
	if err != nil {
		return err
	}

	file, err := store.LoadOrEmpty(path)
	if err != nil {
		return err
	}
	for _, route := range file.Proxies {
		if proxy.NormalizeHost(route.From) == host {
			return fmt.Errorf("%s already exists", host)
		}
	}

	key, err := newKey()
	if err != nil {
		return err
	}
	file.Proxies = append(file.Proxies, store.Route{
		From: host,
		To:   target,
		Key:  key,
	})
	if err := store.Save(path, file); err != nil {
		return err
	}

	fmt.Printf("from: %s\nto: %s\nkey: %s\n", host, target, key)
	return nil
}

func newKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
