// Copyright 2022 Tailscale Inc & Contributors
// SPDX-License-Identifier: BSD-3-Clause

package golink

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Link is the structure stored for each go short link.
type Link struct {
	gorm.Model
	ID       string `gorm:"primaryKey"`
	Short    string // the "foo" part of http://go/foo
	Long     string // the target URL or text/template pattern to run
	Created  time.Time
	LastEdit time.Time // when the link was last edited
	Owner    string    // user@domain
}

type Config struct {
	Host     string // Hostname of the database
	Username string // Username credential to connect to the DB
	Password string // Password credentials to connect to the DB
	Port     int    // Port number of the DB server
}

type Stats struct {
	gorm.Model
	ID      string `gorm:"primaryKey"`
	Created time.Time
	Clicks  int
}

// ClickStats is the number of clicks a set of links have received in a given
// time period. It is keyed by link short name, with values of total clicks.
type ClickStats map[string]int

// Database defines the contract to interact with the links DB
type Database interface {
	LoadAll() ([]*Link, error)
	Load(string) (*Link, error)
	Save(*Link) error
	Delete(string) error
	LoadStats() (ClickStats, error)
	SaveStats(ClickStats) error
	DeleteStats(string) error
}

// linkID returns the normalized ID for a link short name.
func linkID(short string) string {
	id := url.PathEscape(strings.ToLower(short))
	id = strings.ReplaceAll(id, "-", "")
	return id
}

// SQLiteDB stores Links in a SQLite database.
type DB struct {
	db *gorm.DB
	mu sync.RWMutex
}

// NewSQLiteDB returns a new SQLiteDB that stores links in a SQLite database stored at f.
func NewDB(config Config) (*DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=golinks port=%d sslmode=disable", config.Host, config.Username, config.Password, config.Port)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&Link{})

	return newDB(db)
}

func newDB(db *gorm.DB) (*DB, error) {
	return &DB{db: db}, nil
}

// LoadAll returns all stored Links.
//
// The caller owns the returned values.
func (s *DB) LoadAll() ([]*Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var links []*Link
	result := s.db.Find(&links)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return links, err
}

// Load returns a Link by its short name.
//
// It returns fs.ErrNotExist if the link does not exist.
//
// The caller owns the returned value.
func (s *DB) Load(short string) (*Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	link := new(Link)
	result := s.db.First(&link, "id = ?", linkID(short))
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = fs.ErrNotExist
		}
		return nil, err
	}
	return link, nil
}

// Save saves a Link.
func (s *DB) Save(link *Link) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	link.ID = linkID(link.Short)
	result := s.db.Save(&link)
	if err := result.Error; err != nil {
		return err
	}
	rows := result.RowsAffected
	if rows != 1 {
		return fmt.Errorf("expected to affect 1 row, affected %d", rows)
	}
	return nil
}

// Delete removes a Link using its short name.
func (s *DB) Delete(short string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := s.db.Delete(&Link{ID: short})
	if err := result.Error; err != nil {
		return err
	}
	rows := result.RowsAffected
	if rows != 1 {
		return fmt.Errorf("expected to affect 1 row, affected %d", rows)
	}
	return nil
}

// LoadStats returns click stats for links.
func (s *DB) LoadStats() (ClickStats, error) {
	stats := make(ClickStats)
	var clickStats []*Stats

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := s.db.Model(&Stats{}).Select("ID, SUM(Clicks) as Clicks").Group("id").Scan(&clickStats)
	if err := result.Error; err != nil {
		return nil, err
	}

	for _, s := range clickStats {
		stats[s.ID] = s.Clicks
	}

	return stats, nil
}

// SaveStats records click stats for links.  The provided map includes
// incremental clicks that have occurred since the last time SaveStats
// was called.
func (s *DB) SaveStats(stats ClickStats) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	for short, clicks := range stats {
		stat := &Stats{
			ID:     linkID(short),
			Clicks: clicks,
		}
		if err := tx.Save(stat).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// DeleteStats deletes click stats for a link.
func (s *DB) DeleteStats(short string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.db.Delete(&Stats{ID: linkID(short)}).Error; err != nil {
		return err
	}
	return nil
}
