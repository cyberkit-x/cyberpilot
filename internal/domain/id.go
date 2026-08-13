package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ID is a sortable UUIDv7 identifier serialized as its canonical string.
type ID string

func NewID() (ID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("create UUIDv7: %w", err)
	}
	return ID(id.String()), nil
}

func MustNewID() ID {
	id, err := NewID()
	if err != nil {
		panic(err)
	}
	return id
}

func ParseID(value string) (ID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return "", fmt.Errorf("parse id: %w", err)
	}
	if id.Version() != 7 {
		return "", fmt.Errorf("id must be UUIDv7")
	}
	return ID(id.String()), nil
}

func (id ID) Validate() error {
	_, err := ParseID(string(id))
	return err
}

// Timestamp normalizes persisted timestamps to UTC with nanosecond precision.
func Timestamp(value time.Time) time.Time {
	return value.UTC().Round(0)
}
