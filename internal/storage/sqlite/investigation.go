package sqlite

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

func (s *Store) PutRecord(ctx context.Context, session domain.ID, kind string, id domain.ID, value any) error {
	if session.Validate() != nil || id.Validate() != nil || kind == "" {
		return fmt.Errorf("invalid record identity")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO investigation_records(session_id,kind,id,data) VALUES(?,?,?,?) ON CONFLICT(session_id,kind,id) DO UPDATE SET data=excluded.data`, session, kind, id, data)
	return err
}

func (s *Store) Records(ctx context.Context, session domain.ID, kind string) ([]json.RawMessage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM investigation_records WHERE session_id=? AND kind=? ORDER BY id`, session, kind)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var result []json.RawMessage
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		result = append(result, append(json.RawMessage(nil), data...))
	}
	return result, rows.Err()
}
