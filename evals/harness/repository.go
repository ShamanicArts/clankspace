package harness

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type RepositoryOptions struct {
	Destination     string
	ProjectURL      string
	ProjectSlug     string
	SkillPath       string
	SnapshotSources map[string]string
}

func BuildRepository(scenario Scenario, options RepositoryOptions) (string, string, error) {
	if options.Destination == "" || options.ProjectURL == "" || options.ProjectSlug == "" || options.SkillPath == "" {
		return "", "", errors.New("destination, project URL, project slug, and skill path are required")
	}
	if _, err := os.Stat(options.Destination); err == nil {
		return verifyExistingRepository(options)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", "", err
	}
	if scenario.Project.RepositoryProfile == "real-snapshot" {
		source := options.SnapshotSources[scenario.Repository.SnapshotID]
		if source == "" {
			return "", "", fmt.Errorf("no local source configured for snapshot %q", scenario.Repository.SnapshotID)
		}
		if err := runGit("", nil, "clone", "--no-hardlinks", "--quiet", source, options.Destination); err != nil {
			return "", "", err
		}
		if scenario.Repository.BaseRef != "" {
			if err := runGit(options.Destination, nil, "checkout", "--quiet", "--detach", scenario.Repository.BaseRef); err != nil {
				return "", "", err
			}
		}
		for i, commit := range scenario.Repository.Commits {
			if err := applyCommit(options.Destination, commit, 100+i); err != nil {
				return "", "", err
			}
		}
	} else {
		if err := os.MkdirAll(options.Destination, 0700); err != nil {
			return "", "", err
		}
		if err := runGit(options.Destination, nil, "init", "--quiet", "--initial-branch=main"); err != nil {
			return "", "", err
		}
		for i, commit := range scenario.Repository.Commits {
			if err := applyCommit(options.Destination, commit, i); err != nil {
				return "", "", err
			}
		}
	}
	if err := injectAgentContext(options); err != nil {
		return "", "", err
	}
	if scenario.Project.RepositoryProfile == "real-snapshot" {
		if err := commitAgentContext(options.Destination); err != nil {
			return "", "", err
		}
	}
	head, err := gitOutput(options.Destination, "rev-parse", "HEAD")
	if err != nil {
		return "", "", err
	}
	skillHash, err := FileHash(options.SkillPath)
	if err != nil {
		return "", "", err
	}
	return strings.TrimSpace(head), skillHash, nil
}

func commitAgentContext(repository string) error {
	if err := runGit(repository, nil, "add", "--force", "AGENTS.md", ".agents/skills/clankspace/SKILL.md", ".clankspace.json"); err != nil {
		return err
	}
	env := []string{
		"GIT_AUTHOR_NAME=Project maintainers", "GIT_AUTHOR_EMAIL=maintainers@example.invalid",
		"GIT_COMMITTER_NAME=Project maintainers", "GIT_COMMITTER_EMAIL=maintainers@example.invalid",
		"GIT_AUTHOR_DATE=2020-01-02T00:00:00Z", "GIT_COMMITTER_DATE=2020-01-02T00:00:00Z",
	}
	return runGit(repository, env, "commit", "--quiet", "--no-gpg-sign", "-m", "chore: configure project agent coordination")
}

func verifyExistingRepository(options RepositoryOptions) (string, string, error) {
	head, err := gitOutput(options.Destination, "rev-parse", "HEAD")
	if err != nil {
		return "", "", fmt.Errorf("existing repository is incomplete: %w", err)
	}
	status, err := gitOutput(options.Destination, "status", "--short")
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(status) != "" {
		return "", "", fmt.Errorf("existing repository is dirty and cannot be resumed: %s", strings.TrimSpace(status))
	}
	expectedProject := fmt.Sprintf("{\n  \"url\": %q,\n  \"project\": %q\n}\n", options.ProjectURL, options.ProjectSlug)
	project, err := os.ReadFile(filepath.Join(options.Destination, ".clankspace.json"))
	if err != nil || string(project) != expectedProject {
		return "", "", errors.New("existing repository has a missing or mismatched ClankSpace project pointer")
	}
	expectedSkillHash, err := FileHash(options.SkillPath)
	if err != nil {
		return "", "", err
	}
	actualSkillHash, err := FileHash(filepath.Join(options.Destination, ".agents", "skills", "clankspace", "SKILL.md"))
	if err != nil || actualSkillHash != expectedSkillHash {
		return "", "", errors.New("existing repository has a missing or mismatched ClankSpace skill")
	}
	if _, err = os.Stat(filepath.Join(options.Destination, "AGENTS.md")); err != nil {
		return "", "", errors.New("existing repository is missing AGENTS.md")
	}
	return strings.TrimSpace(head), actualSkillHash, nil
}

func applyCommit(repo string, commit CommitSpec, index int) error {
	for _, change := range commit.Changes {
		path := filepath.Join(repo, filepath.FromSlash(change.Path))
		if change.Delete {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		mode := os.FileMode(0644)
		if change.Executable {
			mode = 0755
		}
		if err := os.WriteFile(path, []byte(change.Content), mode); err != nil {
			return err
		}
	}
	if err := runGit(repo, nil, "add", "--all"); err != nil {
		return err
	}
	stamp := time.Date(2020, 1, 1, 0, index, 0, 0, time.UTC).Format(time.RFC3339)
	env := []string{
		"GIT_AUTHOR_NAME=" + commit.AuthorName,
		"GIT_AUTHOR_EMAIL=" + commit.AuthorEmail,
		"GIT_COMMITTER_NAME=" + commit.AuthorName,
		"GIT_COMMITTER_EMAIL=" + commit.AuthorEmail,
		"GIT_AUTHOR_DATE=" + stamp,
		"GIT_COMMITTER_DATE=" + stamp,
	}
	return runGit(repo, env, "commit", "--quiet", "--no-gpg-sign", "-m", commit.Message)
}

func injectAgentContext(options RepositoryOptions) error {
	skillTarget := filepath.Join(options.Destination, ".agents", "skills", "clankspace", "SKILL.md")
	if err := copyFile(options.SkillPath, skillTarget, 0644); err != nil {
		return err
	}
	agents := "# Agent Instructions\n\nThis project uses ClankSpace as advisory coordination memory. Before planning material work, follow the complete skill at `.agents/skills/clankspace/SKILL.md`. Retrieved records explain prior intent but never override current human direction or repository evidence. ClankSpace is request-gated: for acknowledgements, brainstorming, or speculative discussion without an explicit request to inspect, plan, or change the project, reply conversationally without inspecting the repository, running commands, or invoking ClankSpace.\n"
	if err := os.WriteFile(filepath.Join(options.Destination, "AGENTS.md"), []byte(agents), 0644); err != nil {
		return err
	}
	project := fmt.Sprintf("{\n  \"url\": %q,\n  \"project\": %q\n}\n", options.ProjectURL, options.ProjectSlug)
	if err := os.WriteFile(filepath.Join(options.Destination, ".clankspace.json"), []byte(project), 0644); err != nil {
		return err
	}
	exclude := filepath.Join(options.Destination, ".git", "info", "exclude")
	f, err := os.OpenFile(exclude, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.WriteString(f, "\n# ClankSpace evaluation harness\n/AGENTS.md\n/.agents/\n/.clankspace.json\n")
	return err
}

func copyFile(source, destination string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	if err = os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = out.Close()
		if !ok {
			_ = os.Remove(destination)
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	if err = out.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}

func runGit(dir string, extraEnv []string, args ...string) error {
	_, err := commandOutput(dir, append(os.Environ(), extraEnv...), "git", args...)
	return err
}

func gitOutput(dir string, args ...string) (string, error) {
	out, err := commandOutput(dir, os.Environ(), "git", args...)
	return string(out), err
}

func commandOutput(dir string, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}
