// Package storage provides SQLite storage for time data.
package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database connection.
type DB struct {
	conn *sql.DB
}

// TimeRecord represents a stored time measurement.
type TimeRecord struct {
	ID        int64         `json:"id"`
	Server    string        `json:"server"`
	Offset    int64         `json:"offset"`    // Nanoseconds
	Delay     int64         `json:"delay"`     // Nanoseconds
	Jitter    int64         `json:"jitter"`    // Nanoseconds
	Timestamp time.Time     `json:"timestamp"`
	Reachable bool          `json:"reachable"`
	Stratum   uint8         `json:"stratum"`
}

// New creates a new SQLite database connection.
func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.init(); err != nil {
		conn.Close()
		return nil, err
	}

	return db, nil
}

// init creates the required tables.
func (db *DB) init() error {
	schema := `
	CREATE TABLE IF NOT EXISTS time_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		server TEXT NOT NULL,
		offset INTEGER NOT NULL,
		delay INTEGER NOT NULL,
		jitter INTEGER NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		reachable BOOLEAN NOT NULL,
		stratum INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_time_records_server ON time_records(server);
	CREATE INDEX IF NOT EXISTS idx_time_records_timestamp ON time_records(timestamp);
	CREATE INDEX IF NOT EXISTS idx_time_records_server_timestamp ON time_records(server, timestamp);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// StoreRecord saves a time measurement.
func (db *DB) StoreRecord(server string, offset, delay, jitter time.Duration, reachable bool, stratum uint8) error {
	query := `INSERT INTO time_records (server, offset, delay, jitter, reachable, stratum) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, server, offset.Nanoseconds(), delay.Nanoseconds(), jitter.Nanoseconds(), reachable, stratum)
	return err
}

// GetRecentRecords retrieves recent records for a server.
func (db *DB) GetRecentRecords(server string, limit int, since time.Time) ([]TimeRecord, error) {
	query := `SELECT id, server, offset, delay, jitter, timestamp, reachable, stratum 
			  FROM time_records 
			  WHERE server = ? AND timestamp >= ?
			  ORDER BY timestamp DESC LIMIT ?`

	rows, err := db.conn.Query(query, server, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []TimeRecord
	for rows.Next() {
		var r TimeRecord
		var ts string
		err := rows.Scan(&r.ID, &r.Server, &r.Offset, &r.Delay, &r.Jitter, &ts, &r.Reachable, &r.Stratum)
		if err != nil {
			return nil, err
		}
		r.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		records = append(records, r)
	}

	return records, rows.Err()
}

// GetHistory retrieves history for a time range.
func (db *DB) GetHistory(since time.Time) ([]TimeRecord, error) {
	query := `SELECT id, server, offset, delay, jitter, timestamp, reachable, stratum 
			  FROM time_records 
			  WHERE timestamp >= ?
			  ORDER BY timestamp DESC`

	rows, err := db.conn.Query(query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []TimeRecord
	for rows.Next() {
		var r TimeRecord
		var ts string
		err := rows.Scan(&r.ID, &r.Server, &r.Offset, &r.Delay, &r.Jitter, &ts, &r.Reachable, &r.Stratum)
		if err != nil {
			return nil, err
		}
		r.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		records = append(records, r)
	}

	return records, rows.Err()
}

// CleanupOldRecords removes records older than the specified duration.
func (db *DB) CleanupOldRecords(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := db.conn.Exec("DELETE FROM time_records WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}
