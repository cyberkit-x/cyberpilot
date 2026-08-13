package skills

import (
	"strings"
	"testing"
	"testing/fstest"
)

func skill(name, license, body string) fstest.MapFS {
	return fstest.MapFS{name + "/SKILL.md": {Data: []byte("---\nname: " + name + "\ndescription: Use when object identifiers require authorization validation.\nlicense: " + license + "\ndomains: [web, api]\n---\n" + body)}}
}
func TestParseSkillMetadataHashAndProvenance(t *testing.T) {
	doc, err := Parse(skill("validate-object-access", "MIT", "# Validate\nCompare owner and control identities."), "validate-object-access", ParseOptions{Source: "git:https://example.invalid/skills", Version: "abc123", Status: Tested})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Metadata.Name != "validate-object-access" || len(doc.Metadata.Hash) != 64 || doc.Metadata.Status != Tested || !strings.Contains(doc.Metadata.Source, "abc123") || !strings.Contains(string(doc.Instructions), "Compare owner") {
		t.Fatalf("doc=%#v", doc)
	}
}
func TestParseRejectsInvalidSkills(t *testing.T) {
	tests := []struct {
		name, dir  string
		filesystem fstest.MapFS
		contains   string
	}{{"unsafe path", "../skill", skill("skill", "MIT", "body"), "unsafe"}, {"name mismatch", "other", skill("skill", "MIT", "body"), "open"}, {"license", "skill", skill("skill", "unknown", "body"), "license"}, {"empty", "skill", skill("skill", "MIT", "  "), "empty"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.filesystem, tt.dir, ParseOptions{})
			if err == nil || !strings.Contains(err.Error(), tt.contains) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
func TestDuplicateResolutionAndVersionedTrust(t *testing.T) {
	a, _ := Parse(skill("skill", "MIT", "first"), "skill", ParseOptions{Source: "local:a", Version: "1"})
	b, _ := Parse(skill("skill", "MIT", "second"), "skill", ParseOptions{Source: "local:b", Version: "2"})
	accepted, rejected := Resolve([]Document{a, b}, nil)
	if len(accepted) != 0 || rejected["skill"] == nil {
		t.Fatalf("accepted=%v rejected=%v", accepted, rejected)
	}
	accepted, _ = Resolve([]Document{a, b}, map[string]string{"skill": b.Metadata.Source})
	if len(accepted) != 1 || accepted[0].Metadata.Source != b.Metadata.Source {
		t.Fatalf("accepted=%v", accepted)
	}
	current := b.Metadata
	current.Status = Tested
	if got := PreserveStatus(a.Metadata, current); got.Status != Compatible {
		t.Fatalf("status=%s", got.Status)
	}
}
func TestBoundedSkill(t *testing.T) {
	filesystem := skill("skill", "MIT", strings.Repeat("x", MaxSkillBytes))
	if _, err := Parse(filesystem, "skill", ParseOptions{}); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("err=%v", err)
	}
}
