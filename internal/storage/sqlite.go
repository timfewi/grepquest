package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/timfewi/grepquest/internal/game"
	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err = s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS profile (
id INTEGER PRIMARY KEY CHECK (id=1), xp INTEGER NOT NULL, level INTEGER NOT NULL,
solved_json TEXT NOT NULL, mastery_json TEXT NOT NULL, hints_used INTEGER NOT NULL
)`)
	if err != nil {
		return err
	}
	if _, err = s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS ai_cache (cache_key TEXT PRIMARY KEY, response_json BLOB NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		return err
	}
	if _, err = s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS ai_usage (id INTEGER PRIMARY KEY AUTOINCREMENT, used_at TEXT NOT NULL, cost_usd REAL NOT NULL CHECK(cost_usd >= 0))`); err != nil {
		return err
	}
	// Idempotent compatibility migration for profiles created before loot.
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE profile ADD COLUMN inventory_json TEXT NOT NULL DEFAULT '{}'")
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE profile ADD COLUMN loadout_json TEXT NOT NULL DEFAULT '{}'")
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE profile ADD COLUMN reviews_json TEXT NOT NULL DEFAULT '{}'")
	return nil
}

func (s *Store) Load(ctx context.Context) (game.Profile, error) {
	p := game.NewProfile()
	var solved, mastery, inventory, loadout, reviews string
	err := s.db.QueryRowContext(ctx, "SELECT xp, level, solved_json, mastery_json, hints_used, inventory_json, loadout_json,reviews_json FROM profile WHERE id=1").Scan(&p.XP, &p.Level, &solved, &mastery, &p.HintsUsed, &inventory, &loadout, &reviews)
	if err == sql.ErrNoRows {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	if err = json.Unmarshal([]byte(solved), &p.Solved); err != nil {
		return p, fmt.Errorf("decode solved: %w", err)
	}
	if err = json.Unmarshal([]byte(mastery), &p.Mastery); err != nil {
		return p, fmt.Errorf("decode mastery: %w", err)
	}
	if err = json.Unmarshal([]byte(inventory), &p.Inventory); err != nil {
		return p, fmt.Errorf("decode inventory: %w", err)
	}
	if err = json.Unmarshal([]byte(loadout), &p.Loadout); err != nil {
		return p, fmt.Errorf("decode loadout: %w", err)
	}
	if err = json.Unmarshal([]byte(reviews), &p.Reviews); err != nil {
		return p, fmt.Errorf("decode reviews: %w", err)
	}
	return p, nil
}

func (s *Store) Save(ctx context.Context, p game.Profile) error {
	solved, err := json.Marshal(p.Solved)
	if err != nil {
		return err
	}
	mastery, err := json.Marshal(p.Mastery)
	if err != nil {
		return err
	}
	inventory, err := json.Marshal(p.Inventory)
	if err != nil {
		return err
	}
	loadout, err := json.Marshal(p.Loadout)
	if err != nil {
		return err
	}
	reviews, err := json.Marshal(p.Reviews)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO profile(id,xp,level,solved_json,mastery_json,hints_used,inventory_json,loadout_json,reviews_json)
VALUES(1,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET xp=excluded.xp,level=excluded.level,
solved_json=excluded.solved_json,mastery_json=excluded.mastery_json,hints_used=excluded.hints_used,
inventory_json=excluded.inventory_json,loadout_json=excluded.loadout_json,reviews_json=excluded.reviews_json`, p.XP, p.Level, string(solved), string(mastery), p.HintsUsed, string(inventory), string(loadout), string(reviews))
	return err
}
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) GetAI(ctx context.Context, key string) ([]byte, bool, error) {
	var raw []byte
	err := s.db.QueryRowContext(ctx, "SELECT response_json FROM ai_cache WHERE cache_key=?", key).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return raw, true, nil
}
func (s *Store) PutAI(ctx context.Context, key string, raw []byte) error {
	if len(raw) > 64<<10 {
		return fmt.Errorf("AI cache value exceeds 64 KiB")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO ai_cache(cache_key,response_json,created_at) VALUES(?,?,?) ON CONFLICT(cache_key) DO UPDATE SET response_json=excluded.response_json,created_at=excluded.created_at`, key, raw, time.Now().UTC().Format(time.RFC3339))
	return err
}
func (s *Store) AIRequestsToday(ctx context.Context, now time.Time) (int, error) {
	day := now.UTC().Format("2006-01-02")
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM ai_usage WHERE substr(used_at,1,10)=?", day).Scan(&count)
	return count, err
}
func (s *Store) RecordAIRequest(ctx context.Context, now time.Time, cost float64) error {
	if cost < 0 {
		return fmt.Errorf("AI cost cannot be negative")
	}
	_, err := s.db.ExecContext(ctx, "INSERT INTO ai_usage(used_at,cost_usd) VALUES(?,?)", now.UTC().Format(time.RFC3339), cost)
	return err
}
