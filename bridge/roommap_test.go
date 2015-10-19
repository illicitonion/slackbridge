package bridge

import (
	"database/sql"
	"io/ioutil"
	"path"
	"testing"

	"github.com/matrix-org/slackbridge/slack"

	_ "github.com/mattn/go-sqlite3"
)

func TestSlackMessageFilter(t *testing.T) {
	rooms := NewRoomMap(makeDB(t))
	receive := func(ts string) bool {
		return rooms.ShouldNotify(&slack.Message{
			Type:    "message",
			Channel: "CANTINA",
			User:    "U34",
			Text:    "Take more chances",
			TS:      ts,
		})
	}

	if receive("1") {
		t.Fatalf("not linked: should have skipped")
	}

	rooms.Link("!abc123:matrix.org", "CANTINA")

	if !receive("1") {
		t.Fatalf("should have notified")
	}

	if receive("0.5") {
		t.Fatalf("seen newer, should have skipped")
	}

	if !receive("2") {
		t.Fatalf("should have notified")
	}
}

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
	return db
}
