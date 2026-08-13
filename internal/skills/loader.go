package skills

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const MaxSkillBytes = 512 << 10

var validName = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62})$`)

type Document struct {
	Metadata     Metadata
	Instructions []byte
}
type ParseOptions struct {
	Source, Version string
	Status          TrustStatus
}

func Parse(source fs.FS, skillDir string, options ParseOptions) (Document, error) {
	clean := path.Clean(skillDir)
	if clean == "." || clean != skillDir || strings.HasPrefix(clean, "../") || path.IsAbs(clean) {
		return Document{}, errors.New("unsafe skill directory path")
	}
	filePath := path.Join(clean, "SKILL.md")
	file, err := source.Open(filePath)
	if err != nil {
		return Document{}, fmt.Errorf("open %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()
	data, err := readBounded(file, MaxSkillBytes)
	if err != nil {
		return Document{}, err
	}
	front, body, err := splitFrontmatter(data)
	if err != nil {
		return Document{}, err
	}
	var raw struct {
		Name                       string `yaml:"name"`
		Description                string `yaml:"description"`
		License                    string `yaml:"license"`
		Domains, Intents, Requires []string
	}
	decoder := yaml.NewDecoder(bytes.NewReader(front))
	decoder.KnownFields(true)
	if err := decoder.Decode(&raw); err != nil {
		return Document{}, fmt.Errorf("parse skill metadata: %w", err)
	}
	base := path.Base(clean)
	if !validName.MatchString(raw.Name) || raw.Name != base {
		return Document{}, fmt.Errorf("skill name %q must match directory %q and use lowercase letters, digits, and hyphens", raw.Name, base)
	}
	if len(strings.Fields(raw.Description)) < 4 {
		return Document{}, errors.New("skill description must state an actionable trigger and investigation")
	}
	if !validLicense(raw.License) {
		return Document{}, fmt.Errorf("skill %q has missing or unsupported license", raw.Name)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return Document{}, errors.New("skill instructions are empty")
	}
	hash := sha256.Sum256(data)
	status := options.Status
	if status == "" {
		status = Compatible
	}
	metadata := Metadata{Name: raw.Name, Description: strings.TrimSpace(raw.Description), License: raw.License, Source: options.Source + "@" + options.Version, Hash: hex.EncodeToString(hash[:]), Status: status, Domains: raw.Domains, Intents: raw.Intents, Requires: raw.Requires}
	return Document{Metadata: metadata, Instructions: append([]byte(nil), body...)}, nil
}

func Resolve(documents []Document, precedence map[string]string) ([]Document, map[string]error) {
	grouped := map[string][]Document{}
	for _, d := range documents {
		grouped[d.Metadata.Name] = append(grouped[d.Metadata.Name], d)
	}
	var accepted []Document
	rejected := map[string]error{}
	for name, items := range grouped {
		if len(items) == 1 {
			accepted = append(accepted, items[0])
			continue
		}
		preferred := precedence[name]
		matches := 0
		var selected Document
		for _, item := range items {
			if item.Metadata.Source == preferred {
				selected = item
				matches++
			}
		}
		if matches == 1 {
			accepted = append(accepted, selected)
		} else {
			rejected[name] = fmt.Errorf("duplicate skill %q requires explicit source precedence", name)
		}
	}
	sort.Slice(accepted, func(i, j int) bool { return accepted[i].Metadata.Name < accepted[j].Metadata.Name })
	return accepted, rejected
}

func PreserveStatus(previous Metadata, current Metadata) Metadata {
	if previous.Hash != current.Hash && current.Status != Compatible {
		current.Status = Compatible
	}
	return current
}
func validLicense(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "MIT", "APACHE-2.0", "BSD-2-CLAUSE", "BSD-3-CLAUSE", "MPL-2.0", "CC-BY-4.0":
		return true
	}
	return false
}
func splitFrontmatter(data []byte) ([]byte, []byte, error) {
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return nil, nil, errors.New("SKILL.md requires YAML frontmatter")
	}
	rest := data[4:]
	index := bytes.Index(rest, []byte("\n---\n"))
	if index < 0 {
		return nil, nil, errors.New("SKILL.md frontmatter is not terminated")
	}
	return rest[:index], rest[index+5:], nil
}
func readBounded(file fs.File, limit int) ([]byte, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > int64(limit) {
		return nil, fmt.Errorf("SKILL.md exceeds %d byte limit", limit)
	}
	data := make([]byte, info.Size())
	n, err := file.Read(data)
	if err != nil && n != len(data) {
		return nil, err
	}
	return data[:n], nil
}
