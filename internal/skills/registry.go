package skills

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type LocalSource struct{ ID, Root string }

func (s LocalSource) Name() string { return "local:" + s.ID }
func (s LocalSource) FS(context.Context) (fs.FS, error) {
	info, err := os.Stat(s.Root)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("open local skill source: %w", err)
	}
	return os.DirFS(s.Root), nil
}

type GitSource struct{ ID, Checkout, Commit string }

func (s GitSource) Name() string { return "git:" + s.ID }
func (s GitSource) FS(ctx context.Context) (fs.FS, error) {
	if len(s.Commit) < 7 {
		return nil, errors.New("Git skill source requires a pinned commit")
	}
	command := exec.CommandContext(ctx, "git", "-C", s.Checkout, "rev-parse", "HEAD")
	command.Env = []string{"PATH=" + os.Getenv("PATH")}
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("inspect Git skill source: %w", err)
	}
	if strings.TrimSpace(string(output)) != s.Commit {
		return nil, fmt.Errorf("Git skill source is at %s, expected pinned commit %s", strings.TrimSpace(string(output)), s.Commit)
	}
	return os.DirFS(s.Checkout), nil
}

type Index struct {
	Sources    []Source
	Precedence map[string]string
	mu         sync.RWMutex
	documents  map[string]Document
	statuses   map[string]Metadata
	errors     map[string]error
}

func (i *Index) Refresh(ctx context.Context) error {
	var documents []Document
	refreshErrors := map[string]error{}
	for _, source := range i.Sources {
		filesystem, err := source.FS(ctx)
		if err != nil {
			refreshErrors[source.Name()] = err
			continue
		}
		version := sourceVersion(source)
		entries, err := fs.ReadDir(filesystem, ".")
		if err != nil {
			refreshErrors[source.Name()] = err
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			doc, err := Parse(filesystem, entry.Name(), ParseOptions{Source: source.Name(), Version: version})
			if err != nil {
				refreshErrors[source.Name()+"/"+entry.Name()] = err
				continue
			}
			i.mu.RLock()
			previous, ok := i.statuses[doc.Metadata.Name]
			i.mu.RUnlock()
			if ok {
				doc.Metadata = PreserveStatus(previous, doc.Metadata)
			}
			documents = append(documents, doc)
		}
	}
	accepted, duplicates := Resolve(documents, i.Precedence)
	for name, err := range duplicates {
		refreshErrors[name] = err
	}
	next := map[string]Document{}
	statuses := map[string]Metadata{}
	for _, doc := range accepted {
		next[doc.Metadata.Name] = doc
		statuses[doc.Metadata.Name] = doc.Metadata
	}
	i.mu.Lock()
	i.documents, i.statuses, i.errors = next, statuses, refreshErrors
	i.mu.Unlock()
	if len(accepted) == 0 && len(refreshErrors) > 0 {
		return errors.New("no valid skills were loaded")
	}
	return nil
}
func (i *Index) Documents() []Document {
	i.mu.RLock()
	defer i.mu.RUnlock()
	result := make([]Document, 0, len(i.documents))
	for _, doc := range i.documents {
		result = append(result, doc)
	}
	sort.Slice(result, func(a, b int) bool { return result[a].Metadata.Name < result[b].Metadata.Name })
	return result
}
func (i *Index) Errors() map[string]error {
	i.mu.RLock()
	defer i.mu.RUnlock()
	result := map[string]error{}
	for key, value := range i.errors {
		result[key] = value
	}
	return result
}

func (i *Index) Load(_ context.Context, name, hash string) ([]byte, error) {
	i.mu.RLock()
	document, ok := i.documents[name]
	i.mu.RUnlock()
	if !ok {
		return nil, errors.New("skill not found")
	}
	if hash != "" && document.Metadata.Hash != hash {
		return nil, errors.New("skill content hash changed")
	}
	return append([]byte(nil), document.Instructions...), nil
}
func sourceVersion(source Source) string {
	if git, ok := source.(GitSource); ok {
		return git.Commit
	}
	if local, ok := source.(LocalSource); ok {
		absolute, _ := filepath.Abs(local.Root)
		return absolute
	}
	return "configured"
}
