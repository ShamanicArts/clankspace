package harness

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Layout struct {
	Root           string
	ScenarioPath   string
	PreparedPath   string
	RepositoryPath string
	TracesPath     string
	ScoresPath     string
	SecretsPath    string
}

func CreateLayout(root, corpusVersion string, scenario Scenario, canonical []byte) (Layout, string, error) {
	if root == "" || corpusVersion == "" {
		return Layout{}, "", errors.New("ledger root and corpus version are required")
	}
	hash := ContentHash(canonical)
	base := filepath.Join(root, "corpora", corpusVersion, scenario.Split, scenario.ID, hash)
	layout := Layout{
		Root:           base,
		ScenarioPath:   filepath.Join(base, "scenario.json"),
		PreparedPath:   filepath.Join(base, "prepared.json"),
		RepositoryPath: filepath.Join(base, "repo"),
		TracesPath:     filepath.Join(base, "traces"),
		ScoresPath:     filepath.Join(base, "scores"),
		SecretsPath:    filepath.Join(root, "secrets", corpusVersion, hash),
	}
	if existing, err := os.ReadFile(layout.ScenarioPath); err == nil {
		if ContentHash(existing) != hash {
			return Layout{}, "", fmt.Errorf("immutable scenario collision at %s", layout.ScenarioPath)
		}
		return layout, hash, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Layout{}, "", err
	}
	for _, dir := range []string{layout.Root, layout.TracesPath, layout.ScoresPath, layout.SecretsPath} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return Layout{}, "", err
		}
	}
	if err := writeExclusive(layout.ScenarioPath, canonical, 0600); err != nil {
		return Layout{}, "", err
	}
	return layout, hash, nil
}

func WritePrepared(layout Layout, prepared PreparedWorld) error {
	if _, err := os.Stat(layout.PreparedPath); err == nil {
		return fmt.Errorf("prepared artifact already exists: %s", layout.PreparedPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return writeJSONExclusive(layout.PreparedPath, prepared, 0600)
}

func AppendRollout(layout Layout, result RolloutResult) (string, error) {
	path := filepath.Join(layout.TracesPath, result.EpisodeID, "rollout.json")
	if err := writeJSONExclusive(path, result, 0600); err != nil {
		return "", err
	}
	return path, nil
}
