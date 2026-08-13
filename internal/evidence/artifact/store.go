package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

type Store struct {
	root  string
	mu    sync.RWMutex
	index map[domain.ID]metadata
}

type metadata struct {
	Ref  domain.ArtifactRef `json:"ref"`
	Name string             `json:"name"`
}

func Open(root string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(root, "objects"), 0o700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(root, "metadata"), 0o700); err != nil {
		return nil, err
	}
	store := &Store{root: root, index: map[domain.ID]metadata{}}
	entries, err := os.ReadDir(filepath.Join(root, "metadata"))
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, "metadata", entry.Name()))
		if err != nil {
			return nil, err
		}
		var value metadata
		if err := json.Unmarshal(data, &value); err != nil {
			return nil, err
		}
		store.index[value.Ref.ID] = value
	}
	return store, nil
}

func (s *Store) Put(_ context.Context, sessionID domain.ID, mediaType string, protected bool, data []byte) (domain.ArtifactRef, error) {
	if sessionID.Validate() != nil {
		return domain.ArtifactRef{}, errors.New("invalid session id")
	}
	hash := sha256.Sum256(data)
	digest := hex.EncodeToString(hash[:])
	objectPath := filepath.Join(s.root, "objects", digest[:2], digest)
	if err := atomicWriteIfMissing(objectPath, data, 0o600); err != nil {
		return domain.ArtifactRef{}, err
	}
	ref := domain.ArtifactRef{ID: domain.MustNewID(), SessionID: sessionID, SHA256: digest, MediaType: mediaType, Size: int64(len(data)), Protected: protected}
	value := metadata{Ref: ref}
	encoded, _ := json.Marshal(value)
	if err := atomicWrite(filepath.Join(s.root, "metadata", string(ref.ID)+".json"), encoded, 0o600); err != nil {
		return domain.ArtifactRef{}, err
	}
	s.mu.Lock()
	s.index[ref.ID] = value
	s.mu.Unlock()
	return ref, nil
}

func (s *Store) Open(_ context.Context, sessionID, artifactID domain.ID) ([]byte, domain.ArtifactRef, error) {
	s.mu.RLock()
	value, ok := s.index[artifactID]
	s.mu.RUnlock()
	if !ok {
		return nil, domain.ArtifactRef{}, os.ErrNotExist
	}
	if value.Ref.SessionID != sessionID {
		return nil, domain.ArtifactRef{}, fmt.Errorf("artifact belongs to another session")
	}
	path := filepath.Join(s.root, "objects", value.Ref.SHA256[:2], value.Ref.SHA256)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, domain.ArtifactRef{}, err
	}
	hash := sha256.Sum256(data)
	if hex.EncodeToString(hash[:]) != value.Ref.SHA256 {
		return nil, domain.ArtifactRef{}, errors.New("artifact integrity mismatch")
	}
	return data, value.Ref, nil
}

func atomicWriteIfMissing(path string, data []byte, mode os.FileMode) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return atomicWrite(path, data, mode)
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".partial-")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
