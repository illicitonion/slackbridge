package bridge

import (
	"database/sql"
	"io/ioutil"
	"net/http"
	"path"
	"reflect"
	"testing"

	"github.com/matrix-org/slackbridge/matrix"
	"github.com/matrix-org/slackbridge/slack"

	_ "github.com/mattn/go-sqlite3"
)

func TestSlackMessage(t *testing.T) {
	mockMatrixClient := &MockMatrixClient{}
	mockSlackClient := &MockSlackClient{}

	db := makeDB(t)
	rooms, err := NewRoomMap(db)
	if err != nil {
		t.Fatal(err)
	}
	rooms.Link("!abc123:matrix.org", "CANTINA")

	users, err := NewUserMap(db, http.Client{}, rooms)
	if err != nil {
		t.Fatal(err)
	}
	matrixUser := &matrix.User{"@nancy:st.andrews", mockMatrixClient}
	slackUser := &slack.User{"U34", mockSlackClient}
	users.Link(matrixUser, slackUser)

	bridge := Bridge{users, rooms}
	bridge.OnSlackMessage(slack.Message{
		Type:    "message",
		Channel: "CANTINA",
		User:    "U34",
		Text:    "Take more chances",
	})

	want := []call{call{"SendText", []interface{}{"!abc123:matrix.org", "Take more chances"}}}
	if !reflect.DeepEqual(mockMatrixClient.calls, want) {
		t.Fatalf("Wrong Matrix calls, want %v got %v", want, mockMatrixClient.calls)
	}
}

func TestMatrixMessage(t *testing.T) {
	mockMatrixClient := &MockMatrixClient{}
	mockSlackClient := &MockSlackClient{}

	db := makeDB(t)
	rooms, err := NewRoomMap(db)
	if err != nil {
		t.Fatal(err)
	}
	rooms.Link("!abc123:matrix.org", "BOWLINGALLEY")

	users, err := NewUserMap(db, http.Client{}, rooms)
	if err != nil {
		t.Fatal(err)
	}
	matrixUser := &matrix.User{"@sean:st.andrews", mockMatrixClient}
	slackUser := &slack.User{"U35", mockSlackClient}
	users.Link(matrixUser, slackUser)

	bridge := Bridge{users, rooms}
	bridge.OnMatrixRoomMessage(matrix.RoomMessage{
		Type:    "m.room.message",
		Content: []byte(`{"msgtype": "m.text", "body": "It's Nancy!"}`),
		UserID:  "@sean:st.andrews",
		RoomID:  "!abc123:matrix.org",
	})

	want := []call{call{"SendText", []interface{}{"BOWLINGALLEY", "It's Nancy!"}}}
	if !reflect.DeepEqual(mockSlackClient.calls, want) {
		t.Fatalf("Wrong Slack calls, want %v got %v", want, mockSlackClient.calls)
	}
}

func TestLoadsConfig(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	if err != nil {
		t.Fatal(err)
	}
	file := path.Join(dir, "sqlite3.db")
	db := makeDBAt(t, file)
	bridge := makeBridge(t, db)

	slackMessage := &slack.Message{
		Type:    "message",
		Channel: "CANTINA",
		User:    "U34",
		Text:    "Take more chances",
		TS:      "10",
	}
	if !bridge.RoomMap.ShouldNotify(slackMessage) {
		t.Errorf("want should notify, got should not notify")
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Error closing db: %v", err)
	}

	db, err = sql.Open("sqlite3", file)
	if err != nil {
		t.Fatal(err)
	}
	bridge = makeBridge(t, db)
	if bridge.RoomMap.ShouldNotify(slackMessage) {
		t.Errorf("want should not notify, got should notify")
	}
}

func makeBridge(t *testing.T, db *sql.DB) *Bridge {
	mockMatrixClient := &MockMatrixClient{}
	mockSlackClient := &MockSlackClient{}

	rooms, err := NewRoomMap(db)
	if err != nil {
		t.Fatal(err)
	}

	users, err := NewUserMap(db, http.Client{}, rooms)
	if err != nil {
		t.Fatal(err)
	}
	matrixUser := &matrix.User{"@nancy:st.andrews", mockMatrixClient}
	slackUser := &slack.User{"U34", mockSlackClient}
	users.Link(matrixUser, slackUser)

	// Subsequent calls should load link from database, but don't yet
	if err := rooms.Link("!abc123:matrix.org", "CANTINA"); err != nil {
		t.Fatalf("Error linking rooms: %v", err)
	}

	return &Bridge{users, rooms}
}

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

func (m *MockSlackClient) AccessToken() string {
	return ""
}
