package bridge

import (
	"database/sql"
	"io/ioutil"
	"net/http"
	"path"
	"testing"

	"github.com/matrix-org/slackbridge/matrix"
	"github.com/matrix-org/slackbridge/slack"
	_ "github.com/mattn/go-sqlite3"
)

func TestUserMapLoadsConfig(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	if err != nil {
		t.Fatal(err)
	}
	file := path.Join(dir, "sqlite3.db")
	db := makeDBAt(t, file)
	rooms, err := NewRoomMap(db)
	if err != nil {
		t.Fatal(err)
	}
	users, err := NewUserMap(db, http.Client{}, rooms, matrix.NewEchoSuppresser())
	if err != nil {
		t.Fatal(err)
	}
	matrixID := "@foo:somewhere.com"
	slackID := "bar"
	matrixUser := matrix.NewUser(matrixID, &MockMatrixClient{})
	users.Link(matrixUser, &slack.User{slackID, &MockSlackClient{}})
	db.Close()

	db, err = sql.Open("sqlite3", file)
	if err != nil {
		t.Fatal(err)
	}
	rooms, err = NewRoomMap(db)
	if err != nil {
		t.Fatal(err)
	}
	users, err = NewUserMap(db, http.Client{}, rooms, matrix.NewEchoSuppresser())
	if err != nil {
		t.Fatal(err)
	}
	got := users.SlackForMatrix(matrixID)
	if got == nil {
		t.Fatalf("got nil user, want user ID %q", slackID)
	}
	if got.UserID != slackID {
		t.Errorf("want %q got %q", slackID, got.UserID)
	}
}
