package skills

import (
	"context"
	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"os"
	"path/filepath"
	"testing"
)

func TestResourcesRegisteredUntrustedWithoutExecution(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "skill")
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0700); err != nil {
		t.Fatal(err)
	}
	data := []byte("---\nname: skill\ndescription: Use when an API requires focused safe validation.\nlicense: MIT\nrequires: [curl, python]\n---\nInspect evidence.")
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), data, 0600); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(root, "executed")
	script := []byte("#!/bin/sh\ntouch " + marker)
	if err := os.WriteFile(filepath.Join(dir, "scripts", "probe.sh"), script, 0700); err != nil {
		t.Fatal(err)
	}
	index := &Index{Sources: []Source{LocalSource{ID: "fixture", Root: root}}}
	if err := index.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	resources, err := index.Resources(context.Background(), "skill")
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 3 {
		t.Fatalf("resources=%#v", resources)
	}
	for _, resource := range resources {
		if resource.Trusted || resource.SkillHash == "" || resource.SkillSource == "" {
			t.Fatalf("resource=%#v", resource)
		}
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("script executed during indexing")
	}
	var scriptResource Resource
	for _, resource := range resources {
		if resource.Kind == ResourceScript {
			scriptResource = resource
		}
	}
	proposal := domain.ActionProposal{ID: domain.MustNewID(), SessionID: domain.MustNewID(), HypothesisID: domain.MustNewID(), Capability: "runner.exec"}
	action, err := ApprovedResourceAction(scriptResource, proposal)
	if err != nil {
		t.Fatal(err)
	}
	if action.Resource.Hash != scriptResource.Hash || action.Resource.SkillHash != index.Documents()[0].Metadata.Hash {
		t.Fatal("resource identity lost")
	}
}
func TestDependencyCannotBecomeExecutableAction(t *testing.T) {
	_, err := ApprovedResourceAction(Resource{Name: "curl", Kind: ResourceDependency}, domain.ActionProposal{ID: domain.MustNewID(), Capability: "runner.exec"})
	if err == nil {
		t.Fatal("dependency became executable")
	}
}
