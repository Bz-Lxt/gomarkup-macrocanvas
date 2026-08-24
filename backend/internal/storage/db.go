package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/macrocanvas/macrocanvas/internal/clock"
	"github.com/macrocanvas/macrocanvas/internal/macro"

	_ "modernc.org/sqlite"
)

type DB struct {
	sql *sql.DB
}

var ErrMacroNotFound = errors.New("macro not found")

func Open(dir string) (*DB, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	dsn := filepath.Join(dir, "macrocanvas.db")
	sq, err := sql.Open("sqlite", dsn+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	sq.SetMaxOpenConns(1)
	d := &DB{sql: sq}
	if err := d.migrate(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *DB) Close() error { return d.sql.Close() }

func (d *DB) migrate() error {
	_, err := d.sql.Exec(`
CREATE TABLE IF NOT EXISTS macros (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  payload TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS runs (
  id TEXT PRIMARY KEY,
  macro_id TEXT NOT NULL,
  status TEXT NOT NULL,
  payload TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS settings (
  k TEXT PRIMARY KEY,
  v TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS audit (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  at TEXT NOT NULL,
  action TEXT NOT NULL,
  detail TEXT NOT NULL
);
`)
	return err
}

func (d *DB) UpsertMacro(m macro.Macro) error {
	m.UpdatedAt = clock.Format(clock.Now())
	if m.CreatedAt == "" {
		m.CreatedAt = m.UpdatedAt
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = d.sql.Exec(`INSERT INTO macros(id,name,payload,updated_at) VALUES(?,?,?,?)
ON CONFLICT(id) DO UPDATE SET name=excluded.name, payload=excluded.payload, updated_at=excluded.updated_at`,
		m.ID, m.Name, string(b), m.UpdatedAt)
	return err
}

func (d *DB) GetMacro(id string) (macro.Macro, error) {
	var payload string
	err := d.sql.QueryRow(`SELECT payload FROM macros WHERE id=?`, id).Scan(&payload)
	if err != nil {
		return macro.Macro{}, err
	}
	var m macro.Macro
	if err := json.Unmarshal([]byte(payload), &m); err != nil {
		return macro.Macro{}, err
	}
	return m, nil
}

func (d *DB) ListMacros() ([]macro.Macro, error) {
	rows, err := d.sql.Query(`SELECT payload FROM macros ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]macro.Macro, 0)
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var m macro.Macro
		if err := json.Unmarshal([]byte(payload), &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (d *DB) DeleteMacro(id string) error {
	res, err := d.sql.Exec(`DELETE FROM macros WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("delete macro %q: %w", id, ErrMacroNotFound)
	}
	return nil
}

func (d *DB) SaveRun(id, macroID, status string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = d.sql.Exec(`INSERT INTO runs(id,macro_id,status,payload,created_at) VALUES(?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET status=excluded.status, payload=excluded.payload`,
		id, macroID, status, string(b), clock.Format(clock.Now()))
	return err
}

func (d *DB) IncompleteRuns() ([]string, error) {
	rows, err := d.sql.Query(`SELECT id FROM runs WHERE status IN ('queued','running')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, rows.Err()
}

func (d *DB) MarkAbandoned(id string) error {
	_, err := d.sql.Exec(`UPDATE runs SET status='abandoned' WHERE id=? AND status IN ('queued','running')`, id)
	return err
}

func (d *DB) Audit(action, detail string) {
	_, _ = d.sql.Exec(`INSERT INTO audit(at,action,detail) VALUES(?,?,?)`, clock.Format(clock.Now()), action, detail)
}

func (d *DB) SetSetting(k, v string) error {
	_, err := d.sql.Exec(`INSERT INTO settings(k,v) VALUES(?,?) ON CONFLICT(k) DO UPDATE SET v=excluded.v`, k, v)
	return err
}

func (d *DB) GetSetting(k string) (string, bool) {
	var v string
	err := d.sql.QueryRow(`SELECT v FROM settings WHERE k=?`, k).Scan(&v)
	if err != nil {
		return "", false
	}
	return v, true
}

func NewMacroID() string {
	return fmt.Sprintf("m-%d", time.Now().UnixNano())
}
