package harness

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SnapshotResult struct {
	SchemaVersion int       `json:"schemaVersion"`
	ID            string    `json:"id"`
	SourceHead    string    `json:"sourceHead"`
	SnapshotHead  string    `json:"snapshotHead"`
	Repository    string    `json:"repository"`
	SourceURL     string    `json:"sourceUrl"`
	Bundle        string    `json:"bundle"`
	Includes      []string  `json:"includes,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

func CreateSanitizedSnapshot(id, source, ref, destinationRoot string, includes []string) (SnapshotResult, error) {
	if !scenarioIDPattern.MatchString(id) {
		return SnapshotResult{}, errors.New("snapshot id must be a lowercase slug")
	}
	if source == "" || destinationRoot == "" {
		return SnapshotResult{}, errors.New("source and destination root are required")
	}
	if ref == "" {
		ref = "HEAD"
	}
	sourceHead, err := gitOutput(source, "rev-parse", ref+"^{commit}")
	if err != nil {
		return SnapshotResult{}, err
	}
	archiveArgs := []string{"archive", "--format=tar", strings.TrimSpace(sourceHead)}
	if len(includes) > 0 {
		for _, include := range includes {
			if err = validateRelativePath(include); err != nil {
				return SnapshotResult{}, fmt.Errorf("include: %w", err)
			}
		}
		archiveArgs = append(archiveArgs, "--")
		archiveArgs = append(archiveArgs, includes...)
	}
	archive, err := commandOutput(source, os.Environ(), "git", archiveArgs...)
	if err != nil {
		return SnapshotResult{}, err
	}
	repository := filepath.Join(destinationRoot, id)
	bundle := filepath.Join(destinationRoot, id+".bundle")
	if _, err = os.Stat(repository); err == nil {
		return SnapshotResult{}, fmt.Errorf("snapshot repository already exists: %s", repository)
	} else if !errors.Is(err, os.ErrNotExist) {
		return SnapshotResult{}, err
	}
	if err = os.MkdirAll(repository, 0700); err != nil {
		return SnapshotResult{}, err
	}
	if err = extractTrackedArchive(archive, repository); err != nil {
		return SnapshotResult{}, err
	}
	if err = runGit(repository, nil, "init", "--quiet", "--initial-branch=main"); err != nil {
		return SnapshotResult{}, err
	}
	if err = runGit(repository, nil, "add", "--all"); err != nil {
		return SnapshotResult{}, err
	}
	stamp := "2020-01-01T00:00:00Z"
	env := []string{
		"GIT_AUTHOR_NAME=ClankSpace snapshot", "GIT_AUTHOR_EMAIL=snapshot@clankspace.invalid",
		"GIT_COMMITTER_NAME=ClankSpace snapshot", "GIT_COMMITTER_EMAIL=snapshot@clankspace.invalid",
		"GIT_AUTHOR_DATE=" + stamp, "GIT_COMMITTER_DATE=" + stamp,
	}
	message := fmt.Sprintf("snapshot: %s at %s", id, strings.TrimSpace(sourceHead))
	if err = runGit(repository, env, "commit", "--quiet", "--no-gpg-sign", "-m", message); err != nil {
		return SnapshotResult{}, err
	}
	snapshotHead, err := gitOutput(repository, "rev-parse", "HEAD")
	if err != nil {
		return SnapshotResult{}, err
	}
	if _, err = os.Stat(bundle); err == nil {
		return SnapshotResult{}, fmt.Errorf("snapshot bundle already exists: %s", bundle)
	} else if !errors.Is(err, os.ErrNotExist) {
		return SnapshotResult{}, err
	}
	if err = runGit(repository, nil, "bundle", "create", bundle, "HEAD"); err != nil {
		return SnapshotResult{}, err
	}
	sourceURL, urlErr := gitOutput(source, "config", "--get", "remote.origin.url")
	if urlErr != nil {
		sourceURL = ""
	}
	result := SnapshotResult{
		SchemaVersion: 1, ID: id, SourceHead: strings.TrimSpace(sourceHead),
		SnapshotHead: strings.TrimSpace(snapshotHead), Repository: repository,
		SourceURL: strings.TrimSpace(sourceURL), Bundle: bundle, Includes: includes, CreatedAt: time.Now().UTC(),
	}
	if err = writeJSONExclusive(filepath.Join(destinationRoot, id+".snapshot.json"), result, 0600); err != nil {
		return SnapshotResult{}, err
	}
	return result, nil
}

func extractTrackedArchive(body []byte, destination string) error {
	reader := tar.NewReader(bytes.NewReader(body))
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err = validateRelativePath(header.Name); err != nil {
			return err
		}
		target := filepath.Join(destination, filepath.FromSlash(header.Name))
		switch header.Typeflag {
		case tar.TypeXHeader, tar.TypeXGlobalHeader:
			continue
		case tar.TypeDir:
			if err = os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			mode := os.FileMode(header.Mode) & 0777
			if mode == 0 {
				mode = 0644
			}
			file, openErr := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if openErr != nil {
				return openErr
			}
			_, copyErr := io.Copy(file, reader)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		case tar.TypeSymlink:
			if filepath.IsAbs(header.Linkname) {
				return fmt.Errorf("tracked symlink %q has an absolute target", header.Name)
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(target), filepath.FromSlash(header.Linkname)))
			root := filepath.Clean(destination) + string(filepath.Separator)
			if resolved != filepath.Clean(destination) && !strings.HasPrefix(resolved, root) {
				return fmt.Errorf("tracked symlink %q escapes the snapshot", header.Name)
			}
			if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err = os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported tracked archive entry %q (type %d)", header.Name, header.Typeflag)
		}
	}
}
