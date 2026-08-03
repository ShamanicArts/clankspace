package harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func LoadScenario(path string) (Scenario, []byte, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Scenario{}, nil, err
	}
	var scenario Scenario
	if err = json.Unmarshal(body, &scenario); err != nil {
		return Scenario{}, nil, fmt.Errorf("parse scenario: %w", err)
	}
	if err = scenario.Validate(); err != nil {
		return Scenario{}, nil, fmt.Errorf("validate scenario: %w", err)
	}
	canonical, err := json.MarshalIndent(scenario, "", "  ")
	if err != nil {
		return Scenario{}, nil, err
	}
	canonical = append(canonical, '\n')
	return scenario, canonical, nil
}

func ContentHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func FileHash(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return ContentHash(body), nil
}

func writeExclusive(path string, body []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if _, err = f.Write(body); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}

func writeJSONExclusive(path string, value any, mode os.FileMode) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeExclusive(path, append(body, '\n'), mode)
}
