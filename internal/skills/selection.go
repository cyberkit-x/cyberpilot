package skills

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"
)

type Selection struct{ Name, Relevance string }
type Selector interface {
	Select(context.Context, []Candidate) (Selection, error)
}
type Loaded struct {
	Metadata     Metadata
	Relevance    string
	Instructions []byte
	References   map[string][]byte
}

const MaxReferenceBytes = 256 << 10

var markdownLink = regexp.MustCompile(`\[[^\]]+\]\((references/[A-Za-z0-9._/-]+)\)`)

func (i *Index) SelectAndLoad(ctx context.Context, candidates []Candidate, selector Selector, references []string, maxContextBytes int) (Loaded, error) {
	if len(candidates) == 0 {
		return Loaded{}, errors.New("no suitable skill candidates")
	}
	if len(candidates) > 8 {
		return Loaded{}, errors.New("candidate set exceeds selection limit")
	}
	selection, err := selector.Select(ctx, append([]Candidate(nil), candidates...))
	if err != nil {
		return Loaded{}, err
	}
	if strings.TrimSpace(selection.Relevance) == "" {
		return Loaded{}, errors.New("skill selection requires a relevance explanation")
	}
	allowed := false
	for _, candidate := range candidates {
		if candidate.Metadata.Name == selection.Name {
			allowed = true
			break
		}
	}
	if !allowed {
		return Loaded{}, fmt.Errorf("selected skill %q was not in the bounded candidate set", selection.Name)
	}
	i.mu.RLock()
	document, ok := i.documents[selection.Name]
	i.mu.RUnlock()
	if !ok {
		return Loaded{}, errors.New("selected skill is no longer available")
	}
	if maxContextBytes <= 0 {
		maxContextBytes = 512 << 10
	}
	loaded := Loaded{Metadata: document.Metadata, Relevance: selection.Relevance, Instructions: append([]byte(nil), document.Instructions...), References: map[string][]byte{}}
	total := len(loaded.Instructions)
	if total > maxContextBytes {
		return Loaded{}, errors.New("skill body exceeds context budget")
	}
	links := linkedReferences(document.Instructions)
	for _, reference := range references {
		clean := path.Clean(reference)
		if clean != reference || strings.HasPrefix(clean, "../") || !strings.HasPrefix(clean, "references/") || !links[clean] {
			return Loaded{}, fmt.Errorf("reference %q is not a directly linked safe skill reference", reference)
		}
		content, err := i.openReference(ctx, document, clean)
		if err != nil {
			return Loaded{}, err
		}
		total += len(content)
		if total > maxContextBytes {
			return Loaded{}, errors.New("selected skill resources exceed context budget")
		}
		loaded.References[clean] = content
	}
	return loaded, nil
}
func linkedReferences(body []byte) map[string]bool {
	result := map[string]bool{}
	for _, match := range markdownLink.FindAllSubmatch(body, -1) {
		result[string(match[1])] = true
	}
	return result
}
func (i *Index) openReference(ctx context.Context, document Document, reference string) ([]byte, error) {
	for _, source := range i.Sources {
		if !strings.HasPrefix(document.Metadata.Source, source.Name()+"@") {
			continue
		}
		filesystem, err := source.FS(ctx)
		if err != nil {
			return nil, err
		}
		file, err := filesystem.Open(path.Join(document.Metadata.Name, reference))
		if err != nil {
			return nil, err
		}
		defer file.Close()
		return readBounded(file, MaxReferenceBytes)
	}
	return nil, errors.New("skill source is unavailable")
}
func readResource(file fs.File, limit int) ([]byte, error) {
	var buffer bytes.Buffer
	if _, err := buffer.ReadFrom(file); err != nil {
		return nil, err
	}
	if buffer.Len() > limit {
		return nil, errors.New("resource exceeds size limit")
	}
	return buffer.Bytes(), nil
}
