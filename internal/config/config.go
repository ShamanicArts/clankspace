package config

import (
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	Listen             string
	DataDir            string
	DatabasePath       string
	BaseURL            string
	BootstrapToken     string
	WorkspaceName      string
	OwnerName          string
	GitHubToken        string
	AuthMode           string
	InstallationSecret string
	SMTPAddr           string
	SMTPUser           string
	SMTPPassword       string
	SMTPFrom           string
	MailDir            string
	SyncEnabled        bool
	ReplicaName        string
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
		Listen:             listen,
		DataDir:            dataDir,
		DatabasePath:       filepath.Join(dataDir, "clankspace.db"),
		BaseURL:            value("CLANKSPACE_BASE_URL", "http://localhost:8080"),
		BootstrapToken:     os.Getenv("CLANKSPACE_BOOTSTRAP_TOKEN"),
		WorkspaceName:      value("CLANKSPACE_WORKSPACE_NAME", "ClankSpace"),
		OwnerName:          value("CLANKSPACE_OWNER_NAME", "Owner"),
		GitHubToken:        os.Getenv("GITHUB_TOKEN"),
		AuthMode:           value("CLANKSPACE_AUTH_MODE", "bootstrap"),
		InstallationSecret: os.Getenv("CLANKSPACE_INSTALLATION_SECRET"),
		SMTPAddr:           os.Getenv("CLANKSPACE_SMTP_ADDR"),
		SMTPUser:           os.Getenv("CLANKSPACE_SMTP_USER"),
		SMTPPassword:       os.Getenv("CLANKSPACE_SMTP_PASSWORD"),
		SMTPFrom:           os.Getenv("CLANKSPACE_SMTP_FROM"),
		MailDir:            os.Getenv("CLANKSPACE_MAIL_DIR"),
		SyncEnabled:        value("CLANKSPACE_SYNC_ENABLED", "false") == "true",
		ReplicaName:        value("CLANKSPACE_REPLICA_NAME", "ClankSpace"),
	}
	if c.BootstrapToken == "" {
		return c, errors.New("CLANKSPACE_BOOTSTRAP_TOKEN is required")
	}
	if c.AuthMode != "bootstrap" && c.AuthMode != "email" && c.AuthMode != "hybrid" {
		return c, errors.New("CLANKSPACE_AUTH_MODE must be bootstrap, email, or hybrid")
	}
	return c, nil
}

func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
