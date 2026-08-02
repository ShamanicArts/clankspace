package localconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultURL = "http://localhost:8080"

type ProjectFile struct {
	URL     string `json:"url"`
	Project string `json:"project"`
}

type Resolved struct {
	URL             string
	Project         string
	Token           string
	TokenSource     string
	ProjectFilePath string
	CredentialsPath string
}

type credential struct {
	URL     string `json:"url"`
	Project string `json:"project"`
	Token   string `json:"token"`
}

type credentialFile struct {
	Version     int          `json:"version"`
	Credentials []credential `json:"credentials"`
}

func Resolve(start string) (Resolved, error) {
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return Resolved{}, err
		}
	}
	projectFile, projectFilePath, err := findProjectFile(start)
	if err != nil {
		return Resolved{}, err
	}
	resolved := Resolved{
		URL:             first(os.Getenv("CLANKSPACE_URL"), projectFile.URL, defaultURL),
		Project:         first(os.Getenv("CLANKSPACE_PROJECT"), projectFile.Project),
		ProjectFilePath: projectFilePath,
	}
	resolved.CredentialsPath, err = credentialsPath()
	if err != nil {
		return Resolved{}, err
	}
	if token := strings.TrimSpace(os.Getenv("CLANKSPACE_TOKEN")); token != "" {
		resolved.Token = token
		resolved.TokenSource = "environment"
		return resolved, nil
	}
	resolved.Token, err = loadCredential(resolved.CredentialsPath, resolved.URL, resolved.Project)
	if err != nil {
		return Resolved{}, err
	}
	if resolved.Token != "" {
		resolved.TokenSource = "credential_store"
	}
	return resolved, nil
}

func StoreCredential(path, url, project, token string) error {
	url = normalizeURL(url)
	project = strings.TrimSpace(project)
	token = strings.TrimSpace(token)
	if url == "" || project == "" || token == "" {
		return errors.New("url, project, and token are required")
	}
	if path == "" {
		var err error
		path, err = credentialsPath()
		if err != nil {
			return err
		}
	}
	file, err := readCredentials(path)
	if err != nil {
		return err
	}
	found := false
	for i := range file.Credentials {
		if normalizeURL(file.Credentials[i].URL) == url && file.Credentials[i].Project == project {
			file.Credentials[i].Token = token
			found = true
			break
		}
	}
	if !found {
		file.Credentials = append(file.Credentials, credential{URL: url, Project: project, Token: token})
	}
	file.Version = 1
	return writeCredentials(path, file)
}

func CredentialsPath() (string, error) { return credentialsPath() }

func findProjectFile(start string) (ProjectFile, string, error) {
	info, err := os.Stat(start)
	if err == nil && !info.IsDir() {
		start = filepath.Dir(start)
	}
	for dir := filepath.Clean(start); ; dir = filepath.Dir(dir) {
		path := filepath.Join(dir, ".clankspace.json")
		body, err := os.ReadFile(path)
		if err == nil {
			var file ProjectFile
			if err = json.Unmarshal(body, &file); err != nil {
				return ProjectFile{}, "", fmt.Errorf("parse %s: %w", path, err)
			}
			return file, path, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return ProjectFile{}, "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return ProjectFile{}, "", nil
}

func credentialsPath() (string, error) {
	if path := strings.TrimSpace(os.Getenv("CLANKSPACE_CREDENTIALS_FILE")); path != "" {
		return path, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "clankspace", "credentials.json"), nil
}

func loadCredential(path, url, project string) (string, error) {
	if strings.TrimSpace(project) == "" {
		return "", nil
	}
	file, err := readCredentials(path)
	if err != nil {
		return "", err
	}
	url = normalizeURL(url)
	for _, item := range file.Credentials {
		if normalizeURL(item.URL) == url && item.Project == project {
			return strings.TrimSpace(item.Token), nil
		}
	}
	return "", nil
}

func readCredentials(path string) (credentialFile, error) {
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return credentialFile{Version: 1, Credentials: []credential{}}, nil
	}
	if err != nil {
		return credentialFile{}, err
	}
	var file credentialFile
	if err = json.Unmarshal(body, &file); err != nil {
		return credentialFile{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if file.Credentials == nil {
		file.Credentials = []credential{}
	}
	return file, nil
}

func writeCredentials(path string, file credentialFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	body, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".credentials-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.Remove(tmpPath)
		}
	}()
	if err = tmp.Chmod(0600); err != nil {
		return err
	}
	if _, err = tmp.Write(body); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpPath, path); err != nil {
		return err
	}
	ok = true
	return nil
}

func normalizeURL(value string) string { return strings.TrimRight(strings.TrimSpace(value), "/") }

func first(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
