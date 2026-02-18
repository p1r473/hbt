package database

import (
	"path"
	"strings"
	"testing"
	"time"
)

func TestDatabase(t *testing.T) {
	t.Parallel()

	t.Run("Track", testDatabaseTrack)
	t.Run("Hint", testDatabaseHint)
	t.Run("End", testDatabaseEnd)
	t.Run("Delete", testDatabaseDelete)
	t.Run("Save", testDatabaseSave)
	t.Run("Load", testDatabaseLoad)
}

func newDatabase(t *testing.T) *Database {
	t.Helper()

	dbFilename := path.Join(t.TempDir(), strings.ReplaceAll(t.Name(), "/", "_")+".db")

	dbClient, err := newSqliteDB(t.Context(), dbFilename)
	if err != nil {
		t.Fatal(err)
	}

	return &Database{
		db:                     dbClient,
		suggestionStates:       map[string]*suggestionState{},
		suggestionStateTimeout: time.Second,
		lastSessionNodeID:      map[int]int{},
	}
}

func testDatabaseTrack(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	var count int

	err := db.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM node`).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}

	if count != 0 {
		t.Fatalf("expected 0 nodes, got %d", count)
	}

	if err = db.Track(t.Context(), 1, "dir1", "cmd1"); err != nil {
		t.Fatal(err)
	}

	if err = db.Track(t.Context(), 1, "dir2", "cmd1"); err != nil {
		t.Fatal(err)
	}

	err = db.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM node`).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}

	if count != 2 {
		t.Fatalf("expected 2 nodes, got %d", count)
	}
}

func testDatabaseHint(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	got, err := db.Hint(t.Context(), 1, "dir")
	if err != nil {
		t.Fatal(err)
	}

	if got != "" {
		t.Fatalf("expected nothing, got %s", got)
	}

	if err = db.Track(t.Context(), 1, "dir1", "cmd1"); err != nil {
		t.Fatal(err)
	}

	if err = db.Track(t.Context(), 1, "dir1", "cmd2"); err != nil {
		t.Fatal(err)
	}

	if err = db.Track(t.Context(), 1, "dir1", "cmd3"); err != nil {
		t.Fatal(err)
	}

	got, err = db.Hint(t.Context(), 1, "dir1")
	if err != nil {
		t.Fatal(err)
	}

	if got != "cmd1" {
		t.Fatalf("expected cmd1, got %s", got)
	}

	got, err = db.Hint(t.Context(), 1, "dir1")
	if err != nil {
		t.Fatal(err)
	}

	if got != "cmd2" {
		t.Fatalf("expected cmd2, got %s", got)
	}

	got, err = db.Hint(t.Context(), 1, "dir2")
	if err != nil {
		t.Fatal(err)
	}

	if got != "" {
		t.Fatalf("expected nothing, got %s", got)
	}

	time.Sleep(db.suggestionStateTimeout)

	got, err = db.Hint(t.Context(), 1, "dir1")
	if err != nil {
		t.Fatal(err)
	}

	if got != "cmd1" {
		t.Fatalf("expected cmd1, got %s", got)
	}
}

func testDatabaseEnd(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	if err := db.End(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
}

func testDatabaseDelete(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	if err := db.Delete(t.Context(), 1, "wd", "cmd1"); err != nil {
		t.Fatal(err)
	}

	if err := db.Track(t.Context(), 1, "dir1", "cmd1"); err != nil {
		t.Fatal(err)
	}

	var count int

	err := db.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM node`).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Fatalf("expected 1 node, got %d", count)
	}

	if err = db.Delete(t.Context(), 1, "dir1", "cmd1"); err != nil {
		t.Fatal(err)
	}

	err = db.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM node`).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}

	if count != 0 {
		t.Fatalf("expected 0 nodes, got %d", count)
	}
}

func testDatabaseSave(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	if err := db.Save(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
}

func testDatabaseLoad(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	if err := db.Load(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
}
