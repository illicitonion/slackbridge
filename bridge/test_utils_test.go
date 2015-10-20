package bridge

import (
	"database/sql"
	"io/ioutil"
	"path"
	"testing"
)

func makeDB(t *testing.T) *sql.DB {
	dir, err := ioutil.TempDir("", "testdb")
	if err != nil {
		t.Fatal(err)
	}
	return makeDBAt(t, path.Join(dir, "sqlite3.db"))
}

func makeDBAt(t *testing.T, path string) *sql.DB {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("Could not open database: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS rooms(
id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
slack_channel_id TEXT,
matrix_room_id TEXT,
last_slack_timestamp TEXT,
last_matrix_stream_token TEXT
)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS users(
id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
slack_user_id TEXT,
slack_access_token TEXT,
matrix_user_id TEXT,
matrix_access_token TEXT,
matrix_homeserver TEXT
)`); err != nil {
		t.Fatal(err)
	}
	return db
}
