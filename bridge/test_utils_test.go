package bridge

import (
	"database/sql"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
	"testing"

	"github.com/matrix-org/slackbridge/matrix"
)

type call struct {
	method string
	args   []interface{}
}

type MockMatrixClient struct {
	calls []call
}

func (m *MockMatrixClient) SendText(roomID, text string) error {
	m.calls = append(m.calls, call{"SendText", []interface{}{roomID, text}})
	return nil
}

func (m *MockMatrixClient) SendImage(roomID, text string, image *matrix.Image) error {
	m.calls = append(m.calls, call{"SendImage", []interface{}{roomID, text, *image}})
	return nil
}

func (m *MockMatrixClient) JoinRoom(roomID string) error {
	m.calls = append(m.calls, call{"JoinRoom", []interface{}{roomID}})
	return nil
}

func (m *MockMatrixClient) ListRooms() (map[string]bool, error) {
	return nil, nil
}

func (m *MockMatrixClient) AccessToken() string {
	return ""
}

func (m *MockMatrixClient) Homeserver() string {
	return ""
}

type MockSlackClient struct {
	calls []call
}

func (m *MockSlackClient) SendText(channelID, text string) error {
	m.calls = append(m.calls, call{"SendText", []interface{}{channelID, text}})
	return nil
}

func (m *MockSlackClient) SendImage(channelID, fallbackText, imageURL string) error {
	m.calls = append(m.calls, call{"SendImage", []interface{}{channelID, fallbackText, imageURL}})
	return nil
}

func (m *MockSlackClient) AccessToken() string {
	return "slack_access_token"
}

type spyRoundTripper struct {
	fn func(*http.Request) string
}

func (r *spyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := r.fn(req)
	if resp == "" {
		resp = `{"ok": true}`
	}
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(strings.NewReader(resp)),
	}, nil
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
