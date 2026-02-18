// Package database provides a client to a sqlite database containing the
// tracked commands.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/lzambarda/hbt/internal/config"
)

type suggestionState struct {
	index     int
	visitedAt time.Time
}

// Database is client to a sqlite database containing the tracked commands.
type Database struct {
	db                     *sql.DB
	suggestionStates       map[string]*suggestionState
	suggestionStateTimeout time.Duration
	lastSessionNodeID      map[int]int
}

// ErrEmptyFilename is returned when the filename is empty.
var ErrEmptyFilename = errors.New("filename is empty")

// New creates a new database. Filename must be set.\
//
//nolint:varnamelen // Fine here.
func New(ctx context.Context, filename string) (*Database, error) {
	if filename == "" {
		return nil, ErrEmptyFilename
	}

	if config.Debug {
		log.Println("Creating database at", filename)
	}

	//nolint:mnd // It's okay
	// if err := os.MkdirAll(path.Dir(filename), 0o750); err != nil {
	// 	return nil, fmt.Errorf("mkdir %q: %w", path.Dir(filename), err)
	// }

	db, err := newSqliteDB(ctx, filename)
	if err != nil {
		return nil, fmt.Errorf("new sqlite db %q: %w", filename, err)
	}

	return &Database{
		db:                     db,
		suggestionStates:       make(map[string]*suggestionState),
		suggestionStateTimeout: time.Minute,
		lastSessionNodeID:      make(map[int]int),
	}, nil
}

//nolint:varnamelen // Fine here.
func newSqliteDB(ctx context.Context, filename string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, fmt.Errorf("open sqlite3: %w", err)
	}

	// TODO: the `hits` column could be on the edge table for maximum
	// granularity.
	const setupQuery = `
	CREATE TABLE IF NOT EXISTS node (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		directory TEXT NOT NULL,
		command TEXT NOT NULL,
		hits BIGINT NOT NULL DEFAULT 1,
		visited_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (directory, command)
	);

	CREATE TABLE IF NOT EXISTS edge (
		user_id INT NOT NULL,
		from_node_id INT NOT NULL,
		to_node_id INT NOT NULL,
		FOREIGN KEY(from_node_id) REFERENCES node(id),
		FOREIGN KEY(to_node_id) REFERENCES node(id),
		PRIMARY KEY (user_id, from_node_id, to_node_id)
	);`

	_, err = db.ExecContext(ctx, setupQuery)
	if err != nil {
		return nil, fmt.Errorf("unable to create tables: %w", err)
	}

	return db, nil
}

// Track adds to the graph the command cmd performed at path wd by the id
// user/process.
func (d *Database) Track(ctx context.Context, userID int, directory, command string) error {
	// TODO: should take into account the current userID and create edges
	const selectQuery = `SELECT id
		FROM node
		WHERE directory = $1::text AND command = $2::text;`

	var nodeID int

	err := d.db.QueryRowContext(ctx, selectQuery, directory, command).Scan(&nodeID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("unable to check if node exists: %w", err)
		}

		const insertQuery = `INSERT INTO node (directory, command) VALUES ($1::text, $2::text) RETURNING id`

		err = d.db.QueryRowContext(ctx, insertQuery, directory, command).Scan(&nodeID)
		if err != nil {
			return fmt.Errorf("unable to insert node: %w", err)
		}
	}

	const updateQuery = `UPDATE node SET hits = hits + 1 WHERE id=$1::int`

	_, err = d.db.ExecContext(ctx, updateQuery, nodeID)
	if err != nil {
		return fmt.Errorf("unable to update node: %w", err)
	}

	if lastNodeID, ok := d.lastSessionNodeID[userID]; !ok || lastNodeID != nodeID {
		const insertEdgeQuery = `INSERT INTO edge (user_id, from_node_id, to_node_id) VALUES ($1::int, $2::int, $3::int);`

		_, err = d.db.ExecContext(ctx, insertEdgeQuery, userID, lastNodeID, nodeID)
		if err != nil {
			return fmt.Errorf("unable to insert edge: %w", err)
		}

		d.lastSessionNodeID[userID] = nodeID
	}

	return nil
}

// Hint returns the next suggestion for user/process id at path wd.
func (d *Database) Hint(ctx context.Context, userID int, directory string) (string, error) {
	// TODO: should take into account the current userID
	const nodeQuery = `SELECT command FROM node WHERE directory = $1::text ORDER BY hits DESC, id ASC;`

	rows, err := d.db.QueryContext(ctx, nodeQuery, directory)
	if err != nil {
		return "", fmt.Errorf("unable to query node: %w", err)
	}

	defer rows.Close() //nolint:errcheck // Handled below.

	var commands []string

	for rows.Next() {
		var cmd string
		if err := rows.Scan(&cmd); err != nil {
			return "", fmt.Errorf("unable to scan node: %w", err)
		}

		commands = append(commands, cmd)
	}

	rows.Close() //nolint:errcheck,gosec // Handled below.

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error while reading rows: %w", err)
	}

	if len(commands) == 0 {
		return "", nil
	}

	suggestionStateKey := directory // NOTE: This could be user-based

	sst, found := d.suggestionStates[suggestionStateKey]
	if !found {
		d.suggestionStates[suggestionStateKey] = &suggestionState{
			visitedAt: time.Now(),
			index:     0,
		}

		return commands[0], nil
	}

	if time.Since(sst.visitedAt) >= d.suggestionStateTimeout {
		sst.index = 0
	} else {
		sst.index = (sst.index + 1) % len(commands)
	}

	sst.visitedAt = time.Now()

	return commands[sst.index], nil
}

// End clears a session for user/process id. This is useful to reset a
// stateful graph.
func (d *Database) End(_ context.Context, userID int) error {
	delete(d.lastSessionNodeID, userID)

	return nil
}

// Delete removes a previously tracked command. It should not return an
// error.
func (d *Database) Delete(ctx context.Context, _ int, directory, command string) error {
	// TODO: should delete the edge first, and then the node if no other edges
	// are there.
	const query = "DELETE FROM node WHERE directory = $1::text AND command = $2::text;"

	if _, err := d.db.ExecContext(ctx, query, directory, command); err != nil {
		return fmt.Errorf("unable to delete node: %w", err)
	}

	return nil
}

// Save serialises the graph to the given file path.
func (d *Database) Save(_ context.Context, _ string) error {
	// No-op with this implementation
	return nil
}

// Load initialises the graph with a serialiastion at the give file path.
func (d *Database) Load(_ context.Context, _ string) error {
	// No-op with this implementation
	return nil
}
