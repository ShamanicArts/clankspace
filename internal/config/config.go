package config

import (
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	Listen         string
	DataDir        string
	DatabasePath   string
	BaseURL        string
	BootstrapToken string
	WorkspaceName  string
	OwnerName      string
	GitHubToken    string
}

func FromEnv() (Config, error) {
	dataDir := value("CLANKSPACE_DATA_DIR", "./data")
	listen := os.Getenv("CLANKSPACE_LISTEN")
	if listen == "" {
		if port := os.Getenv("PORT"); port != "" {
			listen = ":" + port
		} else {
			listen = ":8080"
		}
	}
	c := Config{
		Listen:         listen,
		DataDir:        dataDir,
		DatabasePath:   filepath.Join(dataDir, "clankspace.db"),
		BaseURL:        value("CLANKSPACE_BASE_URL", "http://localhost:8080"),
		BootstrapToken: os.Getenv("CLANKSPACE_BOOTSTRAP_TOKEN"),
		WorkspaceName:  value("CLANKSPACE_WORKSPACE_NAME", "ClankSpace"),
		OwnerName:      value("CLANKSPACE_OWNER_NAME", "Owner"),
		GitHubToken:    os.Getenv("GITHUB_TOKEN"),
	}
	if c.BootstrapToken == "" {
		return c, errors.New("CLANKSPACE_BOOTSTRAP_TOKEN is required")
	}
	return c, nil
}

func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
