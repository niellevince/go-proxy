package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultPath = "proxies.json"

type File struct {
	HealthHost string  `json:"healthHost"`
	Proxies    []Route `json:"proxies"`
}

type Route struct {
	From string `json:"from"`
	To   string `json:"to"`
	Key  string `json:"key"`
}

func Empty() *File {
	return &File{Proxies: []Route{}}
}

func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if file.Proxies == nil {
		file.Proxies = []Route{}
	}
	return &file, nil
}

func LoadOrEmpty(path string) (*File, error) {
	file, err := Load(path)
	if err == nil {
		return file, nil
	}
	if os.IsNotExist(err) {
		return Empty(), nil
	}
	return nil, err
}

func Save(path string, file *File) error {
	if file.Proxies == nil {
		file.Proxies = []Route{}
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if dir == "" {
		dir = "."
	}
	tmp, err := os.CreateTemp(dir, ".proxies-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	if err := os.Rename(tmpName, path); err != nil {
		// Windows cannot rename over an existing file.
		if rmErr := os.Remove(path); rmErr != nil && !os.IsNotExist(rmErr) {
			os.Remove(tmpName)
			return err
		}
		if err := os.Rename(tmpName, path); err != nil {
			os.Remove(tmpName)
			return err
		}
	}
	return nil
}
