package skills

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

type ResourceKind string

const (
	ResourceScript     ResourceKind = "script"
	ResourceDependency ResourceKind = "dependency"
)

type Resource struct {
	Name, Path                              string
	Kind                                    ResourceKind
	Hash, SkillName, SkillSource, SkillHash string
	Trusted                                 bool
}
type ResourceAction struct {
	Proposal domain.ActionProposal
	Resource Resource
}

func (i *Index) Resources(ctx context.Context, skillName string) ([]Resource, error) {
	i.mu.RLock()
	document, ok := i.documents[skillName]
	i.mu.RUnlock()
	if !ok {
		return nil, errors.New("skill not found")
	}
	var filesystem fs.FS
	for _, source := range i.Sources {
		if strings.HasPrefix(document.Metadata.Source, source.Name()+"@") {
			var err error
			filesystem, err = source.FS(ctx)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	if filesystem == nil {
		return nil, errors.New("skill source unavailable")
	}
	var resources []Resource
	scriptRoot := path.Join(skillName, "scripts")
	_ = fs.WalkDir(filesystem, scriptRoot, func(resourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// A missing optional scripts directory does not invalidate the Skill.
			// Other parse and source errors were already handled above.
			//nolint:nilerr
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		clean := path.Clean(resourcePath)
		if !strings.HasPrefix(clean, scriptRoot+"/") {
			return nil
		}
		data, err := fs.ReadFile(filesystem, clean)
		if err != nil || len(data) > MaxReferenceBytes {
			// Individual optional resources are excluded when unreadable or too
			// large; they are never executed during discovery.
			//nolint:nilerr
			return nil
		}
		sum := sha256.Sum256(data)
		resources = append(resources, Resource{Name: path.Base(clean), Path: strings.TrimPrefix(clean, skillName+"/"), Kind: ResourceScript, Hash: hex.EncodeToString(sum[:]), SkillName: skillName, SkillSource: document.Metadata.Source, SkillHash: document.Metadata.Hash, Trusted: false})
		return nil
	})
	for _, dependency := range document.Metadata.Requires {
		resources = append(resources, Resource{Name: dependency, Kind: ResourceDependency, SkillName: skillName, SkillSource: document.Metadata.Source, SkillHash: document.Metadata.Hash, Trusted: false})
	}
	sort.Slice(resources, func(a, b int) bool {
		if resources[a].Kind == resources[b].Kind {
			return resources[a].Name < resources[b].Name
		}
		return resources[a].Kind < resources[b].Kind
	})
	return resources, nil
}

func ApprovedResourceAction(resource Resource, proposal domain.ActionProposal) (ResourceAction, error) {
	if resource.Trusted {
		return ResourceAction{}, errors.New("skill resources must remain untrusted")
	}
	if resource.Kind != ResourceScript {
		return ResourceAction{}, fmt.Errorf("resource %q is not executable", resource.Name)
	}
	if proposal.ID.Validate() != nil || proposal.Capability != "runner.exec" {
		return ResourceAction{}, errors.New("approved script requires a valid runner.exec action proposal")
	}
	return ResourceAction{Proposal: proposal, Resource: resource}, nil
}
